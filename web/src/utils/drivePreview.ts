import type { DriveEntry, PreviewKind } from '../types/drive'

export function previewKindForEntry(row: DriveEntry): PreviewKind {
  if (row.is_dir) return 'unknown'
  if (row.type === 'image') return 'image'
  if (row.type === 'video') return 'video'
  if (row.type === 'audio') return 'audio'
  if (row.ext?.toLowerCase?.() === 'pdf') return 'pdf'
  return 'unknown'
}

export function isEntryPreviewable(row: DriveEntry): boolean {
  return previewKindForEntry(row) !== 'unknown'
}
