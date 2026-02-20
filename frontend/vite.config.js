import { defineConfig } from "vite";
import vue from '@vitejs/plugin-vue';
import wails from "@wailsio/runtime/plugins/vite";
import tailwindcss from '@tailwindcss/vite'
import path from 'path';

// https://vitejs.dev/config/
export default defineConfig({
  resolve: {
    alias: {
      '@/bindings': path.resolve(__dirname, 'bindings'),
      '@': path.resolve(__dirname, 'src')
    }
  },
  plugins: [tailwindcss(), vue(), wails("./bindings")],
});
