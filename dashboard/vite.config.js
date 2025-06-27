import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ mode }) => ({
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
    emptyOutDir: true,
    // Enhanced development build with debug symbols
    ...(mode === 'development' && {
      minify: false,
      sourcemap: true,
      rollupOptions: {
        output: {
          manualChunks: undefined, // Disable chunking for easier debugging
        }
      }
    }),
    // Production optimizations
    ...(mode === 'production' && {
      minify: 'terser',
      sourcemap: false,
      rollupOptions: {
        output: {
          manualChunks: {
            vendor: ['react', 'react-dom'],
            ui: ['@chakra-ui/react', '@emotion/react', '@emotion/styled'],
            charts: ['recharts'],
            utils: ['axios', 'date-fns']
          }
        }
      }
    })
  }
}))