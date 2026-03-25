import { CF_RECORDS_CACHE_KEY_PREFIX } from './constants'
import type { CfDnsRecord, CachedCfRecordsPayload } from './types'

export function getCacheKey(accountId: number | string, zoneId: string): string {
  return `${CF_RECORDS_CACHE_KEY_PREFIX}${accountId}_${zoneId}`
}

export function getCachedRecords(
  accountId: number | string | null | undefined,
  zoneId: string | null | undefined,
): CfDnsRecord[] | null {
  if (accountId == null || !zoneId) return null

  try {
    const key = getCacheKey(accountId, zoneId)
    const raw = localStorage.getItem(key)
    if (!raw) return null

    const data = JSON.parse(raw) as Partial<CachedCfRecordsPayload>
    return Array.isArray(data?.records) ? data.records : null
  } catch {
    return null
  }
}

export function setCachedRecords(
  accountId: number | string | null | undefined,
  zoneId: string | null | undefined,
  records: CfDnsRecord[] | null | undefined,
): void {
  if (accountId == null || !zoneId) return

  try {
    const key = getCacheKey(accountId, zoneId)
    const data: CachedCfRecordsPayload = { records: records || [], updatedAt: Date.now() }
    localStorage.setItem(key, JSON.stringify(data))
  } catch {
    // Ignore localStorage failures (e.g. quota/security).
  }
}

export function getAllCachedRecordsForAccount(
  accountId: number | string | null | undefined,
): CfDnsRecord[] {
  if (accountId == null) return []

  const prefix = getCacheKey(accountId, '')
  const out: CfDnsRecord[] = []

  try {
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i)
      if (!key || !key.startsWith(prefix)) continue

      const raw = localStorage.getItem(key)
      if (!raw) continue

      const data = JSON.parse(raw) as Partial<CachedCfRecordsPayload>
      if (Array.isArray(data?.records)) out.push(...data.records)
    }
  } catch {
    // Ignore malformed cache entries / localStorage failures.
  }

  return out
}

