import { createContext, useContext, useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { configApi } from '../services/api';

interface AppConfig {
  enableRegistration: boolean;
  enableOAuthRegistration: boolean;
  loading: boolean;
}

const AppConfigContext = createContext<AppConfig>({
  enableRegistration: false,
  enableOAuthRegistration: false,
  loading: true,
});

export const useAppConfig = () => useContext(AppConfigContext);

interface AppConfigProviderProps {
  children: ReactNode;
}

export function AppConfigProvider({ children }: AppConfigProviderProps) {
  const [config, setConfig] = useState<AppConfig>({
    enableRegistration: false,
    enableOAuthRegistration: false,
    loading: true,
  });

  useEffect(() => {
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

    fetchConfig();
  }, []);

  return (
    <AppConfigContext.Provider value={config}>
      {children}
    </AppConfigContext.Provider>
  );
}
