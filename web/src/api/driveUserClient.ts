import api from './client'
import { STORAGE_KEYS } from '../constants/storage'

function driveUserHeaders() {
  const token = (localStorage.getItem(STORAGE_KEYS.driveUserToken) || '').trim()
  const v = token.startsWith('Bearer ') ? token : `Bearer ${token}`
  return { Authorization: v }
}

export async function driveUserListFiles(params: {
  path?: string
  recursive?: boolean
  type?: string
  q?: string
  offset?: number
  limit?: number
}) {
  const res = await api.get('/drive/files', {
    params,
    headers: driveUserHeaders(),
    silentError: true,
  } as any)
  return res.data
}

export async function driveUserListEntries(params: {
  parent_path?: string
  path_prefix?: string
  recursive?: boolean
  type?: string
  q?: string
  sort?: 'name' | 'size' | 'mtime'
  order?: 'asc' | 'desc'
  cursor?: string
  limit?: number
  include_dirs?: boolean
}) {
  const res = await api.get('/drive/entries', {
    params,
    headers: driveUserHeaders(),
    silentError: true,
  } as any)
  return res.data
}

export async function driveUserMkdir(path: string) {
  const res = await api.post(
    '/drive/folders',
    { path },
    {
      headers: driveUserHeaders(),
      silentError: true,
    } as any,
  )
  return res.data
}

export async function driveUserRename(from: string, to: string) {
  const res = await api.post(
    '/drive/rename',
    { from, to },
    {
      headers: driveUserHeaders(),
      silentError: true,
    } as any,
  )
  return res.data
}

export async function driveUserDelete(path: string) {
  const res = await api.delete('/drive/files', {
    params: { path },
    headers: driveUserHeaders(),
    silentError: true,
  } as any)
  return res.data
}

export async function driveUserUpload(
  path: string,
  file: File,
  opts?: { onProgress?: (loaded: number, total?: number) => void },
) {
  const form = new FormData()
  form.append('path', path || '')
  form.append('file', file)
  const res = await api.post('/drive/upload', form, {
    headers: driveUserHeaders(),
    silentError: true,
    onUploadProgress: (evt: any) => {
      if (!opts?.onProgress) return
      opts.onProgress(Number(evt?.loaded) || 0, evt?.total)
    },
  } as any)
  return res.data
}

export async function driveUserDownloadBlob(path: string) {
  const res = await api.get('/drive/download', {
    params: { path },
    headers: driveUserHeaders(),
    responseType: 'blob',
    silentError: true,
  } as any)
  return res.data as Blob
}

export async function driveUserGetPreviewUrl(params: { path: string; expires?: number }) {
  const res = await api.get('/drive/preview-url', {
    params,
    headers: driveUserHeaders(),
    silentError: true,
  } as any)
  return res.data as { ok?: boolean; url?: string; expires_in?: number }
}

