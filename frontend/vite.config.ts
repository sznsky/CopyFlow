import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Vite 配置：开发时代理 API 到后端
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/health': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
