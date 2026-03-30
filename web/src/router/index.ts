import { createRouter, createWebHashHistory } from 'vue-router'
import { ROUTE_PATHS } from '../constants/routes'
import { resolveNavigation } from './guards'

const routes = [
  { path: ROUTE_PATHS.cloudflareLegacy, redirect: ROUTE_PATHS.cloudflareCenter },
  { path: ROUTE_PATHS.ddnsLegacy, redirect: ROUTE_PATHS.cloudflareCenter },
  { path: ROUTE_PATHS.webDevLegacy, redirect: ROUTE_PATHS.fileServices },
  { path: ROUTE_PATHS.smbLegacy, redirect: ROUTE_PATHS.windowsTools },
  {
    path: ROUTE_PATHS.login,
    component: () => import('../layouts/MinimalLayout.vue'),
    children: [{ path: '', name: 'Login', component: () => import('../views/Login.vue') }],
  },
  {
    path: ROUTE_PATHS.driveLogin,
    component: () => import('../layouts/MinimalLayout.vue'),
    children: [{ path: '', name: 'DriveLogin', component: () => import('../views/DriveLogin.vue') }],
  },
  {
    path: ROUTE_PATHS.drivePortal,
    component: () => import('../layouts/MinimalLayout.vue'),
    children: [{ path: '', name: 'DrivePortal', component: () => import('../views/DrivePortal.vue') }],
  },
  {
    path: '/',
    component: () => import('../layouts/AdminLayout.vue'),
    children: [
      { path: '', redirect: { name: 'Dashboard' } },
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('../views/Dashboard.vue'),
        meta: { navLabel: '仪表盘', navOrder: 10 },
      },
      {
        path: 'mappings',
        name: 'Mappings',
        component: () => import('../views/Mappings.vue'),
        meta: { navLabel: '映射', navOrder: 20 },
      },
      {
        path: 'cloudflare-center',
        name: 'CloudflareCenter',
        component: () => import('../views/CloudflareCenter.vue'),
        meta: { navLabel: 'Cloudflare', navOrder: 30 },
      },
      {
        path: 'easytier',
        redirect: () => ({ name: 'WindowsTools', query: { tab: 'easytier' } }),
      },
      {
        path: 'file-services',
        name: 'FileServices',
        component: () => import('../views/FileServices.vue'),
        meta: { navLabel: '文件服务', navOrder: 50 },
      },
      {
        path: 'windows-tools',
        name: 'WindowsTools',
        component: () => import('../views/WindowsTools.vue'),
        meta: { navLabel: 'Windows 工具', navOrder: 55, requiresWindows: true },
      },
      {
        path: 'settings',
        name: 'Settings',
        component: () => import('../views/Settings.vue'),
        meta: { navLabel: '设置', navOrder: 100 },
      },
    ],
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

router.beforeEach((to) => resolveNavigation(to))

export default router
