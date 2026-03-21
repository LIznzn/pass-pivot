import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'node:path'

export default defineConfig(({ command }) => ({
  root: path.resolve(__dirname),
  cacheDir: path.resolve(__dirname, '../node_modules/.vite-portal'),
  envPrefix: 'PPVT_PORTAL_',
  base: command === 'build' ? '/portal/' : '/',
  plugins: [vue()],
  server: {
    host: '0.0.0.0',
    port: 8092
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
