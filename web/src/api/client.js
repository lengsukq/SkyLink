import axios from 'axios'

const client = axios.create({
  baseURL: '/api',
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' },
})

// 若设置了 SKYLINK_ADMIN_SECRET，可在 localStorage 或设置页配置后加到此 header
const token = () => localStorage.getItem('skylink_token') || ''
client.interceptors.request.use((config) => {
  const t = token()
  if (t) config.headers.Authorization = t.startsWith('Bearer ') ? t : `Bearer ${t}`
  return config
})

client.interceptors.response.use(
  (r) => r,
  (e) => {
    if (e.response?.status === 401) {
      localStorage.removeItem('skylink_token')
      if (window.location.hash !== '#/login') {
        window.location.hash = '#/login'
      }
    }
    return Promise.reject(e)
  }
)

export default client
