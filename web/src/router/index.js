import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  { path: '/login', name: 'Login', component: () => import('../views/Login.vue') },
  { path: '/', redirect: '/dashboard' },
  { path: '/dashboard', name: 'Dashboard', component: () => import('../views/Dashboard.vue') },
  { path: '/mappings', name: 'Mappings', component: () => import('../views/Mappings.vue') },
  { path: '/cloudflare-center', name: 'CloudflareCenter', component: () => import('../views/CloudflareCenter.vue') },
  { path: '/cloudflare', redirect: '/cloudflare-center' },
  { path: '/ddns', redirect: '/cloudflare-center' },
  { path: '/easytier', name: 'EasyTier', component: () => import('../views/EasyTier.vue') },
  { path: '/windows-tools', name: 'WindowsTools', component: () => import('../views/WindowsTools.vue') },
  { path: '/webdev', redirect: '/windows-tools' },
  { path: '/smb', redirect: '/windows-tools' },
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
