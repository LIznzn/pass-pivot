import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'node:path'

export default defineConfig({
  root: path.resolve(__dirname),
  cacheDir: path.resolve(__dirname, '../node_modules/.vite-auth'),
  plugins: [vue()],
  server: {
    host: '0.0.0.0',
    port: 8090
  },
  resolve: {
    alias: {
      '@shared': path.resolve(__dirname, '../shared')
    }
  },
  build: {
    outDir: path.resolve(__dirname, 'dist'),
    emptyOutDir: true,
    cssCodeSplit: true,
    rollupOptions: {
      input: {
        auth: path.resolve(__dirname, 'index.html'),
        device: path.resolve(__dirname, 'device.html')
      },
      output: {
        entryFileNames: '[name].js',
        assetFileNames: '[name][extname]'
      }
    }
  }
})
