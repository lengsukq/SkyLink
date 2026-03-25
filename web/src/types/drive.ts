export type DriveEntry = {
  name: string
  path: string
  is_dir: boolean
  size_bytes: number
  modified_at: number
  ext: string
  type: string
}

export type DriveCategory = 'all' | 'image' | 'video' | 'audio' | 'document' | 'archive' | 'other'

export type DriveSort = 'name' | 'size' | 'mtime'

export type DriveOrder = 'asc' | 'desc'

export type DriveViewMode = 'list' | 'grid'

export type PreviewKind = 'image' | 'video' | 'audio' | 'pdf' | 'unknown'

