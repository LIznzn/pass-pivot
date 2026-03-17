import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'node:path'

export default defineConfig({
  root: path.resolve(__dirname),
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
    cssCodeSplit: false,
    rollupOptions: {
      output: {
        inlineDynamicImports: true,
        entryFileNames: 'auth.js',
        assetFileNames: (assetInfo) => {
          if (assetInfo.name?.endsWith('.css')) {
            return 'auth.css'
          }
          return '[name][extname]'
        }
      }
    }
  }
})
