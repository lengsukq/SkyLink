export function formatBytes(n: number | string): string {
  const num = Number(n) || 0
  if (num === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let x = num
  while (x >= 1024 && i < units.length - 1) {
    x /= 1024
    i += 1
  }
  const frac = i === 0 ? 0 : 1
  return `${x.toFixed(frac)} ${units[i]}`
}

export function percentUsed(
  volume: { total_bytes?: number | string; used_bytes?: number | string } | any,
): number {
  const totalBytes = Number(volume?.total_bytes) || 0
  if (totalBytes <= 0) return 0
  const usedBytes = Number(volume?.used_bytes) || 0
  return Math.min(100, Math.round((usedBytes / totalBytes) * 1000) / 10)
}

