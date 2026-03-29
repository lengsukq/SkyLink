import type { DriveCategory, DriveOrder, DriveSort, PreviewKind } from '../types/drive'

export const DRIVE_DEFAULT_LIST_LIMIT = 200
export const DRIVE_UPLOAD_MAX_CONCURRENCY = 3

export const DRIVE_CATEGORY_OPTIONS: Array<{ label: string; key: DriveCategory }> = [
  { label: '全部', key: 'all' },
  { label: '图片', key: 'image' },
  { label: '视频', key: 'video' },
  { label: '音频', key: 'audio' },
  { label: '文档', key: 'document' },
  { label: '压缩包', key: 'archive' },
  { label: '其它', key: 'other' },
]

export const DRIVE_SORT_OPTIONS: Array<{ label: string; value: DriveSort }> = [
  { label: '名称', value: 'name' },
  { label: '大小', value: 'size' },
  { label: '修改时间', value: 'mtime' },
]

export const DRIVE_ORDER_OPTIONS: Array<{ label: string; value: DriveOrder }> = [
  { label: '升序', value: 'asc' },
  { label: '降序', value: 'desc' },
]

export const DRIVE_VIDEO_RATE_OPTIONS: Array<{ label: string; value: number }> = [
  { label: '0.5x', value: 0.5 },
  { label: '1x', value: 1 },
  { label: '1.25x', value: 1.25 },
  { label: '1.5x', value: 1.5 },
  { label: '2x', value: 2 },
]

export const DRIVE_DEFAULT_PREVIEW_KIND: PreviewKind = 'unknown'

