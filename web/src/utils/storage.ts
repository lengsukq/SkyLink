/** 网盘配额等场景使用的「G」为 1024³ 字节（与 formatBytes 中 GB 一致） */
export const BYTES_PER_G = 1024 ** 3

/** 将 G 转为字节（创建/更新配额） */
export function gbToBytes(g: number): number {
  if (!Number.isFinite(g) || g <= 0) return 0
  return Math.round(g * BYTES_PER_G)
}

/** 字节转 G（用于编辑表单，保留两位小数） */
export function bytesToGbForInput(bytes: number): number {
  const n = Number(bytes) || 0
  if (n <= 0) return 0
  return Math.round((n / BYTES_PER_G) * 100) / 100
}

/** 将字节格式化为「x.x G」展示（用于配额列） */
export function formatBytesAsG(bytes: number): string {
  const n = Number(bytes) || 0
  if (n <= 0) return '0 G'
  const g = n / BYTES_PER_G
  if (g >= 100) return `${Math.round(g)} G`
  if (g >= 10) return `${g.toFixed(1)} G`
  return `${g.toFixed(2)} G`
}

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

