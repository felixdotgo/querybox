import naive from 'naive-ui'
import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import { ShowConnectionsWindow } from '@/bindings/github.com/felixdotgo/querybox/services/app'
import App from './App.vue'
import Connections from './views/Connections.vue'
import Home from './views/Home.vue'
import './styles/tailwind.css'

// syntax highlighting styles for document results
import 'highlight.js/styles/github.css'

// Expose an imperative opener for legacy onclicks / global usage
window.openConnectionsWindow = async function openConnectionsWindow() {
  try {
    await ShowConnectionsWindow()
  }
  catch {
    // binding not available — fall back to route change
    window.location.href = '/connections'
  }
}

const routes = [
  { path: '/', component: Home },
  { path: '/connections', component: Connections },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

createApp(App).use(router).use(naive).mount('#app')
