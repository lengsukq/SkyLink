import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  { path: '/login', name: 'Login', component: () => import('../views/Login.vue') },
  { path: '/', redirect: '/dashboard' },
  { path: '/dashboard', name: 'Dashboard', component: () => import('../views/Dashboard.vue') },
  { path: '/mappings', name: 'Mappings', component: () => import('../views/Mappings.vue') },
  { path: '/cloudflare', name: 'Cloudflare', component: () => import('../views/Cloudflare.vue') },
  { path: '/ddns', name: 'DDNS', component: () => import('../views/DDNS.vue') },
  { path: '/settings', name: 'Settings', component: () => import('../views/Settings.vue') },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

router.beforeEach((to) => {
  const token = localStorage.getItem('skylink_token') || ''
  if (to.path === '/login') return true
  if (!token.trim()) return { path: '/login' }
  return true
})

export default router
