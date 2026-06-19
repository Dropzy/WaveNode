import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

const backendPort = process.env.VITE_BACKEND_PORT || '8080'
const backendUrl = `http://127.0.0.1:${backendPort}`
const backendWebSocketUrl = `ws://127.0.0.1:${backendPort}`

// https://vite.dev/config/
export default defineConfig({
  base: './',
  plugins: [react()],
  server: {
    proxy: {
      '/api': backendUrl,
      '/health': backendUrl,
      '/ws': {
        target: backendWebSocketUrl,
        ws: true,
      },
    },
  },
})
