type Cached = { url: string; expiresAt: number }

const cache = new Map<string, Cached>()

export function getCachedPreviewUrl(path: string): string | null {
  const c = cache.get(path)
  if (!c) return null
  if (Date.now() > c.expiresAt) {
    cache.delete(path)
    return null
  }
  return c.url
}

export function setCachedPreviewUrl(path: string, url: string, expiresInSec: number) {
  const ttl = Math.max(30, expiresInSec || 3600) * 1000
  cache.set(path, { url, expiresAt: Date.now() + ttl })
}
