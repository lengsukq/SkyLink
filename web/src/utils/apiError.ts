import axios, { type AxiosError } from 'axios'

function messageFromResponseData(data: unknown): string | null {
  if (data == null || typeof data !== 'object') return null
  const o = data as Record<string, unknown>
  const err = o.error
  const warn = o.warning
  if (typeof err === 'string' && err.trim()) return err
  if (typeof warn === 'string' && warn.trim()) return warn
  return null
}

/**
 * 从 Axios 响应体或 Error 中提取适合展示给用户的一句话。
 */
export function getApiErrorMessage(error: unknown, fallback = 'Request failed'): string {
  if (axios.isAxiosError(error)) {
    const fromData = messageFromResponseData(error.response?.data)
    if (fromData) return fromData
    if (error.message) return error.message
    return fallback
  }
  if (error instanceof Error && error.message) return error.message
  return fallback
}

export function isSilentAxiosError(error: unknown): boolean {
  if (!axios.isAxiosError(error)) return false
  return error.config?.silentError === true
}
