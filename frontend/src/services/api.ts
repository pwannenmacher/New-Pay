import type {
  AuthResponse,
  LoginRequest,
  RegisterRequest,
  PasswordResetRequest,
  PasswordResetConfirm,
  RefreshTokenRequest,
  ApiError,
  Session,
} from '../types';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

// Token storage keys
const ACCESS_TOKEN_KEY = 'access_token';
const REFRESH_TOKEN_KEY = 'refresh_token';

// Token management
export const tokenService = {
  getAccessToken: (): string | null => {
    return localStorage.getItem(ACCESS_TOKEN_KEY);
  },

  setAccessToken: (token: string): void => {
    localStorage.setItem(ACCESS_TOKEN_KEY, token);
  },

  getRefreshToken: (): string | null => {
    return localStorage.getItem(REFRESH_TOKEN_KEY);
  },

  setRefreshToken: (token: string): void => {
    localStorage.setItem(REFRESH_TOKEN_KEY, token);
  },

  clearTokens: (): void => {
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
  },

  setTokens: (accessToken: string, refreshToken: string): void => {
    tokenService.setAccessToken(accessToken);
    tokenService.setRefreshToken(refreshToken);
  },
};

// API client class
class ApiClient {
  private baseUrl: string;
  private isRefreshing = false;
  private refreshSubscribers: ((token: string) => void)[] = [];

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  private async handleResponse<T>(response: Response): Promise<T> {
    if (!response.ok) {
      const error: ApiError = await response.json().catch(() => ({
        error: 'An error occurred',
        message: response.statusText,
      }));
      throw error;
    }

    // Handle 204 No Content
    if (response.status === 204) {
      return {} as T;
    }

    return response.json();
  }

  private async refreshToken(): Promise<string> {
    const response = await fetch(`${this.baseUrl}/auth/refresh`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include', // Important: Include cookies in the request
    });

    if (!response.ok) {
      tokenService.clearTokens();
      window.location.href = '/login';
      throw new Error('Token refresh failed');
    }

    const data: AuthResponse = await response.json();
    tokenService.setTokens(data.access_token, data.refresh_token);
    return data.access_token;
  }

  private onRefreshed(token: string): void {
    this.refreshSubscribers.forEach((callback) => callback(token));
    this.refreshSubscribers = [];
  }

  private addRefreshSubscriber(callback: (token: string) => void): void {
    this.refreshSubscribers.push(callback);
  }

  async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    const token = tokenService.getAccessToken();

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string>),
    };

    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    try {
      const response = await fetch(url, {
        ...options,
        headers,
        credentials: 'include', // Always include cookies
      });

      // If unauthorized and we have a refresh token, try to refresh
      if (response.status === 401 && tokenService.getRefreshToken()) {
        if (!this.isRefreshing) {
          this.isRefreshing = true;

          try {
            const newToken = await this.refreshToken();
            this.isRefreshing = false;
            this.onRefreshed(newToken);

            // Retry original request with new token
            headers['Authorization'] = `Bearer ${newToken}`;
            const retryResponse = await fetch(url, {
              ...options,
              headers,
              credentials: 'include',
            });

            return this.handleResponse<T>(retryResponse);
          } catch (error) {
            this.isRefreshing = false;
            throw error;
          }
        } else {
          // Wait for the token to be refreshed
          return new Promise((resolve, reject) => {
            this.addRefreshSubscriber((token: string) => {
              headers['Authorization'] = `Bearer ${token}`;
              fetch(url, {
                ...options,
                headers,
                credentials: 'include',
              })
                .then((res) => this.handleResponse<T>(res))
                .then(resolve)
                .catch(reject);
            });
          });
        }
      }

      return this.handleResponse<T>(response);
    } catch (error) {
      throw error;
    }
  }

  async get<T>(endpoint: string, options?: RequestInit): Promise<T> {
    return this.request<T>(endpoint, {
      ...options,
      method: 'GET',
    });
  }

  async getPublic<T>(endpoint: string, options?: RequestInit): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    const response = await fetch(url, {
      ...options,
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        ...(options?.headers as Record<string, string>),
      },
    });
    return this.handleResponse<T>(response);
  }

  async post<T>(endpoint: string, data?: unknown, options?: RequestInit): Promise<T> {
    return this.request<T>(endpoint, {
      ...options,
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async put<T>(endpoint: string, data?: unknown, options?: RequestInit): Promise<T> {
    return this.request<T>(endpoint, {
      ...options,
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async delete<T>(endpoint: string, options?: RequestInit): Promise<T> {
    return this.request<T>(endpoint, {
      ...options,
      method: 'DELETE',
    });
  }
}

// Create singleton instance
export const apiClient = new ApiClient(API_BASE_URL);

// Auth API
export const authApi = {
  login: (data: LoginRequest) => apiClient.post<AuthResponse>('/auth/login', data),

  register: (data: RegisterRequest) => apiClient.post<AuthResponse>('/auth/register', data),

  logout: () => apiClient.post<void>('/auth/logout'),

  verifyEmail: (token: string) =>
    apiClient.getPublic<{ message: string }>(`/auth/verify-email?token=${token}`),

  requestPasswordReset: (data: PasswordResetRequest) =>
    apiClient.post<{ message: string }>('/auth/password-reset/request', data),

  confirmPasswordReset: (data: PasswordResetConfirm) =>
    apiClient.post<{ message: string }>('/auth/password-reset/confirm', data),

  refreshToken: (data: RefreshTokenRequest) => apiClient.post<AuthResponse>('/auth/refresh', data),
};

// Session API
export const sessionApi = {
  getMySessions: () => apiClient.get<Session[]>('/users/sessions'),

  deleteMySession: (sessionId: string) =>
    apiClient.delete<{ message: string }>(`/users/sessions/delete?session_id=${sessionId}`),

  deleteAllMySessions: () => apiClient.delete<{ message: string }>('/users/sessions/delete-all'),
};

// Config API
export const configApi = {
  getAppConfig: () =>
    apiClient.getPublic<{ enable_registration: boolean; enable_oauth_registration: boolean }>(
      '/config/app'
    ),

  getOAuthConfig: () =>
    apiClient.getPublic<{ enabled: boolean; providers: { name: string }[] }>('/config/oauth'),
};

export default apiClient;
