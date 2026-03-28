import axios, { type AxiosInstance, type AxiosError } from 'axios'
import { notifyError } from '../ui/notify'
import { ROUTE_PATHS } from '../constants/routes'
import { STORAGE_KEYS } from '../constants/storage'

const client: AxiosInstance = axios.create({
  baseURL: '/api',
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' },
})

function shouldNotifyGlobally(error: unknown): boolean {
  return (error as any)?.config?.silentError !== true
}

function adminToken(): string {
  return localStorage.getItem(STORAGE_KEYS.skylinkToken) || ''
}

function driveUserToken(): string {
  return localStorage.getItem(STORAGE_KEYS.driveUserToken) || ''
}

/**
 * 将请求路径规范为「相对 /api」的形式，例如 /drive/files。
 * 避免 baseURL、查询串导致误判。
 */
function apiRequestPath(config: { url?: string }): string {
  let u = String(config.url || '')
  if (!u) return ''
  if (u.startsWith('http')) {
    try {
      u = new URL(u).pathname
    } catch {
      return u
    }
  }
  const q = u.indexOf('?')
  if (q >= 0) u = u.slice(0, q)
  if (u.startsWith('/api/')) u = u.slice(4)
  if (!u.startsWith('/')) u = `/${u}`
  return u
}

function isDriveAuthLoginPath(p: string): boolean {
  return p === '/drive/auth/login'
}

function isDriveAccountsAdminPath(p: string): boolean {
  return p.startsWith('/drive/accounts')
}

/** 个人网盘用户 JWT 接口（与管理员 Bearer 完全隔离） */
function isDriveUserAPIPath(p: string): boolean {
  if (!p.startsWith('/drive/')) return false
  if (isDriveAuthLoginPath(p)) return false
  if (isDriveAccountsAdminPath(p)) return false
  return true
}

client.interceptors.request.use((config) => {
  // 默认 Content-Type: application/json 会破坏 multipart：须让浏览器带上带 boundary 的 multipart/form-data
  if (config.data instanceof FormData && config.headers) {
    if (typeof config.headers.delete === 'function') {
      config.headers.delete('Content-Type')
    } else {
      delete (config.headers as Record<string, unknown>)['Content-Type']
    }
  }

  const path = apiRequestPath(config)
  const configAny = config as any
  const headers = (configAny.headers ??= {}) as Record<string, unknown>

  // 网盘用户登录：匿名，禁止附带管理员 Token（避免与网盘会话混淆）
  if (isDriveAuthLoginPath(path)) {
    delete headers.Authorization
    return config
  }

  // 子账号管理：仅管理员密码 / Token
  if (isDriveAccountsAdminPath(path)) {
    const t = adminToken()
    if (t) headers.Authorization = t.startsWith('Bearer ') ? t : `Bearer ${t}`
    else delete headers.Authorization
    return config
  }

  // 个人网盘文件/预览 URL 等：仅网盘 JWT，禁止使用管理员凭证
  if (isDriveUserAPIPath(path)) {
    const dt = driveUserToken()
    if (dt) headers.Authorization = dt.startsWith('Bearer ') ? dt : `Bearer ${dt}`
    else delete headers.Authorization
    return config
  }

  const t = adminToken()
  if (!t) return config
  headers.Authorization = t.startsWith('Bearer ') ? t : `Bearer ${t}`
  return config
})

client.interceptors.response.use(
  (r) => r,
  (e: AxiosError) => {
    const status = e.response?.status
    const data = e.response?.data as any
    const msg =
      (typeof data?.error === 'string' && data.error) ||
      (typeof data?.warning === 'string' && data.warning) ||
      e.message ||
      'Request failed'

    if (e.response?.status === 401) {
      const path = apiRequestPath(e.config || {})

      if (isDriveUserAPIPath(path)) {
        localStorage.removeItem(STORAGE_KEYS.driveUserToken)
        if (shouldNotifyGlobally(e)) {
          notifyError('未授权', msg)
        }
        const driveLoginHash = `#${ROUTE_PATHS.driveLogin}`
        if (window.location.hash !== driveLoginHash) {
          window.location.hash = driveLoginHash
        }
        return Promise.reject(e)
      }

      localStorage.removeItem(STORAGE_KEYS.skylinkToken)
      if (shouldNotifyGlobally(e)) {
        notifyError('未授权', msg)
      }
      const loginHash = `#${ROUTE_PATHS.login}`
      if (window.location.hash !== loginHash) {
        window.location.hash = loginHash
      }
      return Promise.reject(e)
    }

    if (shouldNotifyGlobally(e)) {
      if (status) {
        notifyError(`请求失败 (${status})`, msg)
      } else {
        notifyError('网络错误', msg)
      }
    }

    return Promise.reject(e)
  },
)

export default client
