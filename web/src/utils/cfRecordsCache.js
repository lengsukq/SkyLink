const CACHE_KEY_PREFIX = 'skylink_cf_records_'

export function getCacheKey(accountId, zoneId) {
  return `${CACHE_KEY_PREFIX}${accountId}_${zoneId}`
}

/**
 * @param {number|string|null} accountId
 * @param {string|null} zoneId
 * @returns {Array|null} records or null if missing/invalid
 */
export function getCachedRecords(accountId, zoneId) {
  if (accountId == null || !zoneId) return null
  try {
    const key = getCacheKey(accountId, zoneId)
    const raw = localStorage.getItem(key)
    if (!raw) return null
    const data = JSON.parse(raw)
    return Array.isArray(data?.records) ? data.records : null
  } catch {
    return null
  }
}

/**
 * @param {number|string|null} accountId
 * @param {string|null} zoneId
 * @param {Array} records
 */
export function setCachedRecords(accountId, zoneId, records) {
  if (accountId == null || !zoneId) return
  try {
    const key = getCacheKey(accountId, zoneId)
    const data = { records: records || [], updatedAt: Date.now() }
    localStorage.setItem(key, JSON.stringify(data))
  } catch (_) {}
}

/**
 * Merge all cached records for the given account (all zones).
 * @param {number|string|null} accountId
 * @returns {Array}
 */
export function getAllCachedRecordsForAccount(accountId) {
  if (accountId == null) return []
  const prefix = getCacheKey(accountId, '')
  const out = []
  try {
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i)
      if (!key || !key.startsWith(prefix)) continue
      const raw = localStorage.getItem(key)
      if (!raw) continue
      const data = JSON.parse(raw)
      if (Array.isArray(data?.records)) out.push(...data.records)
    }
  } catch (_) {}
  return out
}
