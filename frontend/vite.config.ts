import { defineConfig, loadEnv, type ProxyOptions } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig(({ mode }) => {
  // The dev server proxies the API to the running Compose stack (Caddy) so the
  // browser keeps a single origin for the session cookie. The backend only
  // accepts mutations from TF_ALLOWED_ORIGIN, so Origin/Referer are rewritten.
  const apiTarget = loadEnv(mode, process.cwd(), '').VITE_API_TARGET || 'http://localhost:8080'
  const proxy: ProxyOptions = {
    target: apiTarget,
    changeOrigin: true,
    configure: (server) => {
      server.on('proxyReq', (request) => {
        if (request.getHeader('origin')) request.setHeader('origin', apiTarget)
        const referer = request.getHeader('referer')
        if (typeof referer === 'string') request.setHeader('referer', new URL(new URL(referer).pathname, apiTarget).href)
      })
    },
  }

  return {
    plugins: [vue(), tailwindcss()],
    server: {
      port: 5173,
      strictPort: true,
      proxy: { '/api': proxy, '/health': proxy },
    },
  }
})
