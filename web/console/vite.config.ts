import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'node:path'

export default defineConfig(({ command }) => ({
  root: path.resolve(__dirname),
  cacheDir: path.resolve(__dirname, '../node_modules/.vite-console'),
  envPrefix: 'PPVT_CONSOLE_',
  base: command === 'build' ? '/console/' : '/',
  plugins: [
    vue(),
    {
      name: 'console-root-redirect',
      configureServer(server) {
        server.middlewares.use((req, res, next) => {
          if (req.url === '/') {
            res.statusCode = 302
            res.setHeader('Location', '/console')
            res.end()
            return
          }
          next()
        })
      },
      configurePreviewServer(server) {
        server.middlewares.use((req, res, next) => {
          if (req.url === '/') {
            res.statusCode = 302
            res.setHeader('Location', '/console')
            res.end()
            return
          }
          next()
        })
      }
    }
  ],
  server: {
    host: '0.0.0.0',
    port: 8093
  },
  resolve: {
    alias: {
      '@shared': path.resolve(__dirname, '../shared')
    }
  },
  build: {
    outDir: path.resolve(__dirname, 'dist'),
    emptyOutDir: true
  }
}))
