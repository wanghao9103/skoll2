import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    host: '0.0.0.0',
    port: 5173,
    proxy: {
      '/api': {
        target: process.env.VITE_PROXY_TARGET || 'http://localhost:18080',
        changeOrigin: true
      },
      '/health': {
        target: process.env.VITE_PROXY_TARGET || 'http://localhost:18080',
        changeOrigin: true
      }
    }
  }
})
