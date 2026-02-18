import { createContext, useContext, useState, useEffect, useRef, type ReactNode } from 'react';
import type { User, LoginRequest, RegisterRequest } from '../types';
import { authApi, tokenService, apiClient } from '../services/api';
import { useAppConfig } from './AppConfigContext';
import { getTimeUntilRefresh, isTokenExpired } from '../utils/jwtUtils';

interface AuthContextType {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (credentials: LoginRequest) => Promise<void>;
  register: (data: RegisterRequest) => Promise<void>;
  logout: () => Promise<void>;
  updateUser: (user: User) => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider = ({ children }: AuthProviderProps) => {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const { refetch: refetchAppConfig } = useAppConfig();
  const refreshTimerRef = useRef<number | null>(null);
  const sessionCheckTimerRef = useRef<number | null>(null);

  useEffect(() => {
    // Check if user is already logged in on mount
    const initAuth = async () => {
      const token = tokenService.getAccessToken();

      if (token) {
        try {
          // Fetch current user profile
          const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || '/api/v1';
          const response = await fetch(`${apiBaseUrl}/users/profile`, {
            headers: {
              Authorization: `Bearer ${token}`,
            },
            credentials: 'include',
          });

          if (response.ok) {
            const userData = await response.json();
            setUser(userData);
            setupTokenRefresh();
            setupSessionCheck();
          } else {
            tokenService.clearTokens();
          }
        } catch (error) {
          console.error('Failed to fetch user:', error);
          tokenService.clearTokens();
        }
      }

      setIsLoading(false);
    };

    initAuth();

    // Cleanup timers on unmount
    return () => {
      if (refreshTimerRef.current) {
        clearTimeout(refreshTimerRef.current);
      }
      if (sessionCheckTimerRef.current) {
        clearInterval(sessionCheckTimerRef.current);
      }
    };
  }, []);

  // Setup automatic token refresh (1 minute before expiration)
  const setupTokenRefresh = () => {
    // Clear existing timer
    if (refreshTimerRef.current) {
      clearTimeout(refreshTimerRef.current);
    }

    const token = tokenService.getAccessToken();
    if (!token) return;

    const timeUntilRefresh = getTimeUntilRefresh(token);
    if (timeUntilRefresh === null) return;

    console.log(`Token will be refreshed in ${Math.floor(timeUntilRefresh / 1000)} seconds`);

    refreshTimerRef.current = setTimeout(async () => {
      try {
        const response = await apiClient.post<{ access_token: string; refresh_token: string }>(
          '/auth/refresh',
          {}
        );
        tokenService.setTokens(response.access_token, response.refresh_token);
        console.log('Token refreshed successfully');
        // Setup next refresh
        setupTokenRefresh();
      } catch (error) {
        console.error('Failed to refresh token:', error);
        // Token refresh failed, logout user
        tokenService.clearTokens();
        setUser(null);
        window.location.href = '/login';
      }
    }, timeUntilRefresh);
  };

  // Setup session check (every 30 seconds)
  const setupSessionCheck = () => {
    // Clear existing timer
    if (sessionCheckTimerRef.current) {
      clearInterval(sessionCheckTimerRef.current);
    }

    sessionCheckTimerRef.current = setInterval(async () => {
      const token = tokenService.getAccessToken();

      if (!token) {
        console.log('No token found, redirecting to login');
        clearInterval(sessionCheckTimerRef.current!);
        setUser(null);
        window.location.href = '/login';
        return;
      }

      // Check token expiration locally first (fast check)
      if (isTokenExpired(token)) {
        console.log('Token expired, redirecting to login');
        clearInterval(sessionCheckTimerRef.current!);
        tokenService.clearTokens();
        setUser(null);
        window.location.href = '/login';
        return;
      }

      // Validate session on server (checks if session was revoked)
      try {
        await authApi.validateSession();
      } catch (error) {
        console.log('Session validation failed, redirecting to login');
        clearInterval(sessionCheckTimerRef.current!);
        tokenService.clearTokens();
        setUser(null);
        window.location.href = '/login';
      }
    }, 30000); // Check every 30 seconds
  };

  const login = async (credentials: LoginRequest) => {
    try {
      const response = await authApi.login(credentials);
      tokenService.setTokens(response.access_token, response.refresh_token);
      setUser(response.user);
      setupTokenRefresh();
      setupSessionCheck();
    } catch (error) {
      throw error;
    }
  };

  const register = async (data: RegisterRequest) => {
    try {
      const response = await authApi.register(data);
      tokenService.setTokens(response.access_token, response.refresh_token);
      setUser(response.user);
      setupTokenRefresh();
      setupSessionCheck();
    } catch (error) {
      throw error;
    }
  };

  const logout = async () => {
    // Clear timers
    if (refreshTimerRef.current) {
      clearTimeout(refreshTimerRef.current);
    }
    if (sessionCheckTimerRef.current) {
      clearInterval(sessionCheckTimerRef.current);
    }

    try {
      await authApi.logout();
    } catch (error) {
      console.error('Logout error:', error);
    } finally {
      tokenService.clearTokens();
      setUser(null);
      // Refetch app config after logout to update registration availability
      await refetchAppConfig();
    }
  };

  const updateUser = (updatedUser: User) => {
    setUser(updatedUser);
  };

  const value: AuthContextType = {
    user,
    isLoading,
    isAuthenticated: !!user,
    login,
    register,
    logout,
    updateUser,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = () => {
  const context = useContext(AuthContext);

  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }

  return context;
};
