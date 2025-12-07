import { useState, useEffect } from 'react';
import { apiClient } from '../services/api';

export interface OAuthProvider {
  name: string;
}

export interface OAuthConfig {
  enabled: boolean;
  providers: OAuthProvider[];
}

export const useOAuthConfig = () => {
  const [config, setConfig] = useState<OAuthConfig | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const loadConfig = async () => {
      try {
        const data = await apiClient.get<OAuthConfig>('/config/oauth');
        setConfig(data);
      } catch (error) {
        console.error('Failed to load OAuth config:', error);
        setConfig({ enabled: false, providers: [] });
      } finally {
        setIsLoading(false);
      }
    };

    loadConfig();
  }, []);

  return { config, isLoading };
};
