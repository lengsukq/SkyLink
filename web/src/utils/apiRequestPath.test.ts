import { describe, expect, it } from 'vitest'
import { apiRequestPath } from './apiRequestPath'

describe('apiRequestPath', () => {
  it('normalizes relative path and strips query', () => {
    expect(apiRequestPath({ url: '/drive/files?x=1' })).toBe('/drive/files')
  })

  it('strips /api prefix', () => {
    expect(apiRequestPath({ url: '/api/drive/entries' })).toBe('/drive/entries')
  })

  it('parses pathname from absolute URL', () => {
    expect(apiRequestPath({ url: 'http://localhost:5173/api/stats' })).toBe('/stats')
  })

  it('returns empty for missing url', () => {
    expect(apiRequestPath({})).toBe('')
  })
})
