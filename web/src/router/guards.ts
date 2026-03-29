import type { RouteLocationNormalized, RouteLocationRaw } from 'vue-router'
import { ROUTE_PATHS } from '../constants/routes'
import { STORAGE_KEYS } from '../constants/storage'

export function resolveNavigation(to: RouteLocationNormalized): boolean | RouteLocationRaw {
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
}
