import { createContext, useContext, useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { useMantineColorScheme } from '@mantine/core';

type ThemeMode = 'light' | 'dark' | 'auto';

interface ThemeContextType {
  themeMode: ThemeMode;
  toggleTheme: (mode: ThemeMode) => void;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

export const useTheme = () => {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within ThemeProvider');
  }
  return context;
};

interface ThemeProviderProps {
  children: ReactNode;
}

export function ThemeProvider({ children }: ThemeProviderProps) {
  const { setColorScheme } = useMantineColorScheme();
  const [themeMode, setThemeMode] = useState<ThemeMode>(() => {
    // Lade gespeicherte Präferenz aus localStorage
    const saved = localStorage.getItem('themeMode');
    return (saved as ThemeMode) || 'auto';
  });

  useEffect(() => {
    // Speichere Präferenz
    localStorage.setItem('themeMode', themeMode);

    if (themeMode === 'auto') {
      // Verwende System-Präferenz
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
      const handleChange = (e: MediaQueryListEvent | MediaQueryList) => {
        setColorScheme(e.matches ? 'dark' : 'light');
      };

      // Setze initialen Wert
      setColorScheme(mediaQuery.matches ? 'dark' : 'light');

      // Höre auf Änderungen
      mediaQuery.addEventListener('change', handleChange);

      return () => {
        mediaQuery.removeEventListener('change', handleChange);
      };
    } else {
      // Verwende manuelle Einstellung
      setColorScheme(themeMode);
    }
  }, [themeMode, setColorScheme]);

  const toggleTheme = (mode: ThemeMode) => {
    setThemeMode(mode);
  };

  return (
    <ThemeContext.Provider value={{ themeMode, toggleTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}
