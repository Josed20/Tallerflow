import { defineConfig, loadEnv, type ProxyOptions } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { readFileSync } from 'node:fs'

const packageVersion: string = JSON.parse(readFileSync(new URL('./package.json', import.meta.url), 'utf8')).version

export default defineConfig(({ mode }) => {
  // The dev server proxies the API to the running Compose stack (Caddy) so the
  // browser keeps a single origin for the session cookie. The backend only
  // accepts mutations from TF_ALLOWED_ORIGIN, so Origin/Referer are rewritten.
  const buildEnv = loadEnv(mode, process.cwd(), '')
  const apiTarget = buildEnv.VITE_API_TARGET || 'http://localhost:8080'
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
    define: { 'import.meta.env.VITE_APP_VERSION': JSON.stringify(buildEnv.VITE_APP_VERSION || packageVersion) },
    plugins: [vue(), tailwindcss()],
    server: {
      port: 5173,
      strictPort: true,
      proxy: { '/api': proxy, '/health': proxy },
    },
  }
})
