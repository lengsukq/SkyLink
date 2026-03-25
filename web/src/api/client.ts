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

// 登录成功后会将管理密码保存在 localStorage，并作为 Authorization Bearer 发送
function token(): string {
  return localStorage.getItem(STORAGE_KEYS.skylinkToken) || ''
}

client.interceptors.request.use((config) => {
  const t = token()
  if (!t) return config

  // Axios 的 headers 类型在不同适配器下会变化，这里用宽松写法避免大量断言。
  const configAny = config as any
  const headers = (configAny.headers ??= {}) as Record<string, unknown>
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

