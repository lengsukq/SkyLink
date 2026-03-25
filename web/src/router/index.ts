import { createRouter, createWebHashHistory } from 'vue-router'
import { ROUTE_PATHS } from '../constants/routes'
import { STORAGE_KEYS } from '../constants/storage'

const routes = [
  { path: ROUTE_PATHS.login, name: 'Login', component: () => import('../views/Login.vue') },
  { path: ROUTE_PATHS.driveLogin, name: 'DriveLogin', component: () => import('../views/DriveLogin.vue') },
  { path: ROUTE_PATHS.drivePortal, name: 'DrivePortal', component: () => import('../views/DrivePortal.vue') },
  { path: ROUTE_PATHS.root, redirect: ROUTE_PATHS.dashboard },
  { path: ROUTE_PATHS.dashboard, name: 'Dashboard', component: () => import('../views/Dashboard.vue') },
  { path: ROUTE_PATHS.mappings, name: 'Mappings', component: () => import('../views/Mappings.vue') },
  {
    path: ROUTE_PATHS.cloudflareCenter,
    name: 'CloudflareCenter',
    component: () => import('../views/CloudflareCenter.vue'),
  },
  { path: ROUTE_PATHS.cloudflareLegacy, redirect: ROUTE_PATHS.cloudflareCenter },
  { path: ROUTE_PATHS.ddnsLegacy, redirect: ROUTE_PATHS.cloudflareCenter },
  { path: ROUTE_PATHS.easyTier, name: 'EasyTier', component: () => import('../views/EasyTier.vue') },
  { path: ROUTE_PATHS.fileServices, name: 'FileServices', component: () => import('../views/FileServices.vue') },
  { path: ROUTE_PATHS.windowsTools, name: 'WindowsTools', component: () => import('../views/WindowsTools.vue') },
  { path: ROUTE_PATHS.webDevLegacy, redirect: ROUTE_PATHS.fileServices },
  { path: ROUTE_PATHS.smbLegacy, redirect: ROUTE_PATHS.windowsTools },
  { path: ROUTE_PATHS.settings, name: 'Settings', component: () => import('../views/Settings.vue') },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

router.beforeEach((to) => {
  // Drive user portal is independent from admin auth.
  if (to.path === ROUTE_PATHS.driveLogin) return true
  if (to.path === ROUTE_PATHS.drivePortal) {
    const driveToken = localStorage.getItem(STORAGE_KEYS.driveUserToken) || ''
    if (!driveToken.trim()) return { path: ROUTE_PATHS.driveLogin }
    return true
  }

  const token = localStorage.getItem(STORAGE_KEYS.skylinkToken) || ''
  if (to.path === ROUTE_PATHS.login) return true
  if (!token.trim()) return { path: ROUTE_PATHS.login }
  return true
})

export default router

