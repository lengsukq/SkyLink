import { describe, expect, it } from 'vitest'
import axios from 'axios'
import { getApiErrorMessage, isSilentAxiosError } from './apiError'

describe('getApiErrorMessage', () => {
  it('reads error string from Axios response data', () => {
    const err = new axios.AxiosError('fail')
    err.response = {
      status: 400,
      statusText: 'Bad Request',
      data: { error: 'bad request' },
      headers: {},
      config: {} as any,
    }
    expect(getApiErrorMessage(err)).toBe('bad request')
  })

  it('prefers error over warning in response data', () => {
    const err = new axios.AxiosError('x')
    err.response = {
      status: 400,
      statusText: 'Bad Request',
      data: { error: 'e', warning: 'w' },
      headers: {},
      config: {} as any,
    }
    expect(getApiErrorMessage(err)).toBe('e')
  })

  it('uses warning when error absent', () => {
    const err = new axios.AxiosError('x')
    err.response = {
      status: 400,
      statusText: 'Bad Request',
      data: { warning: 'heads up' },
      headers: {},
      config: {} as any,
    }
    expect(getApiErrorMessage(err)).toBe('heads up')
  })

  it('falls back to message and custom fallback', () => {
    const err = new axios.AxiosError('network down')
    expect(getApiErrorMessage(err)).toBe('network down')
    expect(getApiErrorMessage(new Error('oops'))).toBe('oops')
    expect(getApiErrorMessage('x')).toBe('Request failed')
    expect(getApiErrorMessage('x', 'custom')).toBe('custom')
  })
})

describe('isSilentAxiosError', () => {
  it('returns true when config.silentError is set', () => {
    const err = new axios.AxiosError('x')
    err.config = { silentError: true } as any
    expect(isSilentAxiosError(err)).toBe(true)
  })

  it('returns false for normal Axios errors', () => {
    const err = new axios.AxiosError('x')
    err.config = {} as any
    expect(isSilentAxiosError(err)).toBe(false)
  })
})
