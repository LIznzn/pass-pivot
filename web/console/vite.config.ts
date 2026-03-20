import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'node:path'

export default defineConfig(({ command }) => ({
  root: path.resolve(__dirname),
  envPrefix: 'PPVT_CONSOLE_',
  base: command === 'build' ? '/console/' : '/',
  plugins: [vue()],
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
