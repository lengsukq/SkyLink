import axios, {
  type AxiosInstance,
  type AxiosError,
  type InternalAxiosRequestConfig,
} from 'axios'
import { notifyError } from '../ui/notify'
import { ROUTE_PATHS } from '../constants/routes'
import { STORAGE_KEYS } from '../constants/storage'
import { apiRequestPath } from '../utils/apiRequestPath'
import { getApiErrorMessage, isSilentAxiosError } from '../utils/apiError'

const client: AxiosInstance = axios.create({
  baseURL: '/api',
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' },
})

function shouldNotifyGlobally(error: unknown): boolean {
  return !isSilentAxiosError(error)
}

function adminToken(): string {
  return localStorage.getItem(STORAGE_KEYS.skylinkToken) || ''
}

function driveUserToken(): string {
  return localStorage.getItem(STORAGE_KEYS.driveUserToken) || ''
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

function headerBag(config: InternalAxiosRequestConfig): Record<string, unknown> {
  if (!config.headers) {
    config.headers = {} as InternalAxiosRequestConfig['headers']
  }
  return config.headers as unknown as Record<string, unknown>
}

client.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  // 默认 Content-Type: application/json 会破坏 multipart：须让浏览器带上带 boundary 的 multipart/form-data
  if (config.data instanceof FormData && config.headers) {
    if (typeof config.headers.delete === 'function') {
      config.headers.delete('Content-Type')
    } else {
      delete headerBag(config)['Content-Type']
    }
  }

  const path = apiRequestPath(config)
  const headers = headerBag(config)

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
    const msg = getApiErrorMessage(e)

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
export { apiRequestPath } from '../utils/apiRequestPath'
