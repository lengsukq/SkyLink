import { describe, expect, it } from 'vitest'
import { formatBytes, percentUsed } from './storage'

describe('storage utils', () => {
  it('formats bytes with human readable unit', () => {
    expect(formatBytes(0)).toBe('0 B')
    expect(formatBytes(1024)).toBe('1.0 KB')
    expect(formatBytes(1536)).toBe('1.5 KB')
  })

  it('calculates percent used with clamped upper bound', () => {
    expect(percentUsed({ total_bytes: 0, used_bytes: 10 })).toBe(0)
    expect(percentUsed({ total_bytes: 200, used_bytes: 50 })).toBe(25)
    expect(percentUsed({ total_bytes: 200, used_bytes: 300 })).toBe(100)
  })
})
