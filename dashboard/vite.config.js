import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  base: '/_dashboard/',
  server: {
    port: 3001,
    proxy: {
      // Proxy API calls directly to go-deployd server
      '/todos': {
        target: 'http://localhost:2403',
        changeOrigin: true
      },
      '/_admin': {
        target: 'http://localhost:2403',
        changeOrigin: true
      }
    }
  },
  build: {
    outDir: '../web/dashboard',
    emptyOutDir: true
  }
})