/**
 * 将请求路径规范为「相对 /api」的形式，例如 /drive/files。
 * 避免 baseURL、查询串导致误判。
 */
export function apiRequestPath(config: { url?: string }): string {
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
