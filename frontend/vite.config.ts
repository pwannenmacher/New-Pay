import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    host: true,
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: (id) => {
          if (id.includes('node_modules')) {
            if (id.includes('react') || id.includes('react-dom') || id.includes('react-router-dom')) {
              return 'react-vendor';
            }
            if (id.includes('@mantine/core') || id.includes('@mantine/hooks')) {
              return 'mantine-core';
            }
            if (id.includes('@mantine/form') || id.includes('@mantine/notifications')) {
              return 'mantine-extras';
            }
          }
        },
      },
    },
    chunkSizeWarningLimit: 600,
  },
})
