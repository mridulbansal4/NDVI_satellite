import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:5000',
        changeOrigin: true,
        timeout: 300_000,
      },
      '/chatbot': {
        target: 'http://127.0.0.1:5000',
        changeOrigin: true,
        timeout: 120_000,
      },
    },
  },
})
