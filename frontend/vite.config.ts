import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
      '^/admin/services': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/health': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '^/services/register': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '^/services/heartbeat': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
