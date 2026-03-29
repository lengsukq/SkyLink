import type { AxiosProgressEvent } from 'axios'
import { getCachedPreviewUrl, setCachedPreviewUrl } from '../utils/drivePreviewUrlCache'
import api from './client'
// 网盘请求使用与管理员相同的 axios 实例；Authorization 由 client 拦截器按路径自动注入「网盘 JWT」（见 client.ts）。

const silent = { silentError: true as const }

/**
 * 调用 GET /api/drive/files：直接遍历目录的列表（offset/limit）。
 * 内置 Web 个人网盘（DriveUserBrowserPanel）使用 {@link driveUserListEntries}（/drive/entries，索引 + cursor）；
 * 本函数保留给脚本、第三方或未来场景，当前仓库内无其它引用。
 */
export async function driveUserListFiles(params: {
  path?: string
  recursive?: boolean
  type?: string
  q?: string
  offset?: number
  limit?: number
}) {
  const res = await api.get('/drive/files', { ...silent, params })
  return res.data
}

/** 调用 GET /api/drive/entries：网盘 UI 主用的列表（排序、递归搜索、cursor 分页、类型筛选）。 */
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
  const res = await api.get('/drive/entries', { ...silent, params })
  return res.data
}

export async function driveUserMkdir(path: string) {
  const res = await api.post('/drive/folders', { path }, silent)
  return res.data
}

export async function driveUserRename(from: string, to: string) {
  const res = await api.post('/drive/rename', { from, to }, silent)
  return res.data
}

export async function driveUserDelete(path: string) {
  const res = await api.delete('/drive/files', { ...silent, params: { path } })
  return res.data
}

// 预留：若后端增加「复制文件」能力，可在此封装 POST /api/drive/copy 并在 UI 中对接。

export async function driveUserUpload(
  path: string,
  file: File,
  opts?: { onProgress?: (loaded: number, total?: number) => void },
) {
  const form = new FormData()
  form.append('path', path || '')
  form.append('file', file)
  const res = await api.post('/drive/upload', form, {
    ...silent,
    onUploadProgress: (evt: AxiosProgressEvent) => {
      if (!opts?.onProgress) return
      opts.onProgress(Number(evt.loaded) || 0, evt.total)
    },
  })
  return res.data
}

export async function driveUserDownloadBlob(path: string) {
  const res = await api.get('/drive/download', {
    ...silent,
    params: { path },
    responseType: 'blob',
  })
  return res.data as Blob
}

export async function driveUserGetPreviewUrl(params: { path: string; expires?: number }) {
  const res = await api.get('/drive/preview-url', { ...silent, params })
  return res.data as { ok?: boolean; url?: string; expires_in?: number }
}

/** Uses in-memory cache to avoid duplicate preview-url requests (grid + modal). */
export async function driveUserGetPreviewUrlCached(params: { path: string; expires?: number }) {
  const hit = getCachedPreviewUrl(params.path)
  if (hit) return { url: hit, expires_in: 0 as number | undefined }
  const data = await driveUserGetPreviewUrl(params)
  if (data.url) setCachedPreviewUrl(params.path, String(data.url), Number(data.expires_in) || 3600)
  return data
}

export type DriveUserIndexStatus = {
  running: boolean
  started_at: number
  finished_at: number
  scanned_files: number
  scanned_dirs: number
  last_error: string
}

/** 重建当前账号的网盘索引（与磁盘对齐）；完成后需重新拉取列表。 */
export async function driveUserIndexRebuild() {
  const res = await api.post('/drive/index/rebuild', {}, silent)
  return res.data as { ok?: boolean }
}

export async function driveUserIndexStatus() {
  const res = await api.get('/drive/index/status', silent)
  return res.data as { status?: DriveUserIndexStatus }
}
