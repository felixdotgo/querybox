import naive from 'naive-ui'
import { createApp } from 'vue'
import { createRouter, createWebHashHistory } from 'vue-router'
import { ShowConnectionsWindow, ShowPluginsWindow } from '@/bindings/github.com/felixdotgo/querybox/services/app'
import App from './App.vue'
import Connections from './views/Connections.vue'
import Home from './views/Home.vue'
import Plugins from './views/Plugins.vue'
import './styles/tailwind.css'

// syntax highlighting styles for document results
import 'highlight.js/styles/github.css'

// Expose imperative openers for legacy onclicks / global usage
window.openConnectionsWindow = async function openConnectionsWindow() {
  try {
    await ShowConnectionsWindow()
  }
  catch {
    // binding not available — fall back to route change
    window.location.href = '/#/connections'
  }
}
window.openPluginsWindow = async function openPluginsWindow() {
  try {
    await ShowPluginsWindow()
  }
  catch {
    window.location.href = '/#/plugins'
  }
}

const routes = [
  { path: '/', component: Home },
  { path: '/connections', component: Connections },
  { path: '/plugins', component: Plugins },
]

const router = createRouter({
  history: createWebHashHistory(import.meta.env.BASE_URL),
  routes,
})

createApp(App).use(router).use(naive).mount('#app')
