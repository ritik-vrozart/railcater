import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const devPort = Number(env.VITE_DEV_PORT || 5173)
  const apiProxy = env.VITE_API_PROXY_TARGET || 'http://localhost:8080'

  return {
    plugins: [react(), tailwindcss()],
    server: {
      port: devPort,
      proxy: {
        '/api': {
          target: apiProxy,
          changeOrigin: true,
        },
      },
    },
  }
})
