import { createContext, useContext, useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { configApi } from '../services/api';

interface AppConfig {
  enableRegistration: boolean;
  enableOAuthRegistration: boolean;
  loading: boolean;
  refetch: () => Promise<void>;
}

const AppConfigContext = createContext<AppConfig>({
  enableRegistration: false,
  enableOAuthRegistration: false,
  loading: true,
  refetch: async () => {},
});

export const useAppConfig = () => useContext(AppConfigContext);

interface AppConfigProviderProps {
  children: ReactNode;
}

export function AppConfigProvider({ children }: AppConfigProviderProps) {
  const [config, setConfig] = useState<Omit<AppConfig, 'refetch'>>({
    enableRegistration: false,
    enableOAuthRegistration: false,
    loading: true,
  });

  const fetchConfig = async () => {
    try {
      const response = await configApi.getAppConfig();
      setConfig({
        enableRegistration: response.enable_registration,
        enableOAuthRegistration: response.enable_oauth_registration,
        loading: false,
      });
    } catch (error) {
      console.error('Failed to fetch app config:', error);
      // Default to disabled if config fetch fails
      setConfig({
        enableRegistration: false,
        enableOAuthRegistration: false,
        loading: false,
      });
    }
  };

  useEffect(() => {
    fetchConfig();
  }, []);

  return (
    <AppConfigContext.Provider value={{ ...config, refetch: fetchConfig }}>
      {children}
    </AppConfigContext.Provider>
  );
}
