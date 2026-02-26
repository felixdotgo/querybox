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
  // make sure the dev server listens on the loopback address rather than
  // relying on the ambiguous "localhost" hostname.  Wails will attempt to
  // connect to `http://localhost:<port>` when running `wails dev`; on Windows
  // the OS may resolve `localhost` to an IPv6 address (`::1`) while Vite
  // binds only to IPv4, which results in connection failures like the one
  // reported by the user.  Hardcoding `127.0.0.1` avoids that mismatch.
  server: {
    host: '127.0.0.1',
    // Wails writes built assets into frontend/dist while running in dev mode.
    // Without this ignore pattern the vite watcher sees the new files and
    // triggers an immediate reload, which can race with the backend startup
    // and lead to the "unable to connect to frontend server" error.  We
    // simply ignore the dist directory entirely during development.
    watch: {
      ignored: ['**/dist/**'],
    },
  },
  plugins: [tailwindcss(), vue(), wails('./bindings')],
})
