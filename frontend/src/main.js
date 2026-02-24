import './styles/tailwind.css';
// syntax highlighting styles for document results
import 'highlight.js/styles/github.css';
import { createApp } from 'vue';
import { createRouter, createWebHistory } from 'vue-router';
import App from './App.vue';
import Home from './views/Home.vue';
import Connections from './views/Connections.vue';
import { ShowConnectionsWindow } from "@/bindings/github.com/felixdotgo/querybox/services/app";
import naive from 'naive-ui';

// Expose an imperative opener for legacy onclicks / global usage
window.openConnectionsWindow = async function openConnectionsWindow() {
  try {
    await ShowConnectionsWindow();
    return;
  } catch (err) {
    // binding not available — fall back to route change
    window.location.href = '/connections';
  }
};

const routes = [
  { path: '/', component: Home },
  { path: '/connections', component: Connections },
];

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
});

createApp(App).use(router).use(naive).mount('#app');

