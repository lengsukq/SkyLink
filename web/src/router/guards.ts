import type { RouteLocationNormalized, RouteLocationRaw } from 'vue-router'
import { ROUTE_PATHS } from '../constants/routes'
import { STORAGE_KEYS } from '../constants/storage'

function adminToken(): string {
  return localStorage.getItem(STORAGE_KEYS.skylinkToken) || ''
}

function driveToken(): string {
  return localStorage.getItem(STORAGE_KEYS.driveUserToken) || ''
}

/** 公开登录页，不要求已有会话 */
function isPublicAuthRoute(path: string): boolean {
  return path === ROUTE_PATHS.login || path === ROUTE_PATHS.driveLogin
}

/**
 * 需要管理员会话的路由：除根路径、两种登录页、网盘门户外的均为控制台路由。
 * 缺少管理员 token 时只应去管理员登录页（非默认的网盘登录页）。
 */
function requiresAdminSession(path: string): boolean {
  if (isPublicAuthRoute(path)) return false
  if (path === ROUTE_PATHS.drivePortal) return false
  if (path === ROUTE_PATHS.root) return false
  return true
}

export function resolveNavigation(to: RouteLocationNormalized): boolean | RouteLocationRaw {
  if (to.path === ROUTE_PATHS.driveLogin) return true

  if (to.path === ROUTE_PATHS.drivePortal) {
    if (!driveToken().trim()) return { path: ROUTE_PATHS.driveLogin }
    return true
  }

  if (to.path === ROUTE_PATHS.root) {
    if (adminToken().trim()) return true
    if (driveToken().trim()) return { path: ROUTE_PATHS.drivePortal }
    return { path: ROUTE_PATHS.driveLogin }
  }

  if (to.path === ROUTE_PATHS.login) return true

  if (requiresAdminSession(to.path) && !adminToken().trim()) {
    return { path: ROUTE_PATHS.login }
  }

  return true
}
