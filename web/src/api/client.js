import axios from 'axios'
import { notifyError } from '../ui/notify'

const client = axios.create({
  baseURL: '/api',
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' },
})

// 登录成功后会将管理密码保存在 localStorage，并作为 Authorization Bearer 发送
const token = () => localStorage.getItem('skylink_token') || ''
client.interceptors.request.use((config) => {
  const t = token()
  if (t) config.headers.Authorization = t.startsWith('Bearer ') ? t : `Bearer ${t}`
  return config
})

client.interceptors.response.use(
  (r) => r,
  (e) => {
    const status = e.response?.status
    const data = e.response?.data
    const msg =
      (typeof data?.error === 'string' && data.error) ||
      (typeof data?.warning === 'string' && data.warning) ||
      e.message ||
      'Request failed'

    if (e.response?.status === 401) {
      localStorage.removeItem('skylink_token')
      notifyError('未授权', msg)
      if (window.location.hash !== '#/login') {
        window.location.hash = '#/login'
      }
      return Promise.reject(e)
    }

    if (status) {
      notifyError(`请求失败 (${status})`, msg)
    } else {
      notifyError('网络错误', msg)
    }
    return Promise.reject(e)
  }
)

export default client
