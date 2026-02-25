import path from 'node:path'
import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import wails from '@wailsio/runtime/plugins/vite'
import { defineConfig } from 'vite'

// https://vitejs.dev/config/
export default defineConfig({
  resolve: {
    alias: {
      '@/bindings': path.resolve(__dirname, 'bindings'),
      '@': path.resolve(__dirname, 'src'),
    },
  },
  plugins: [tailwindcss(), vue(), wails('./bindings')],
})
