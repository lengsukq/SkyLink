import { computed, type Ref } from 'vue'
import type { DriveEntry } from '../../types/drive'
import { isEntryPreviewable } from '../../utils/drivePreview'

export function useDrivePreviewNav(rows: Ref<DriveEntry[]>, previewItem: Ref<DriveEntry | null>, openPreview: (item: DriveEntry) => void) {
  const previewableRows = computed(() => rows.value.filter((r) => isEntryPreviewable(r)))

  const previewNavMeta = computed(() => {
    const list = previewableRows.value
    const p = previewItem.value?.path
    if (!p || !list.length) return { total: 0, index: 0 }
    const idx = list.findIndex((r) => r.path === p)
    return { total: list.length, index: idx >= 0 ? idx : 0 }
  })

  function previewNavPrev() {
    const list = previewableRows.value
    const i = list.findIndex((r) => r.path === previewItem.value?.path)
    if (i <= 0) return
    openPreview(list[i - 1])
  }

  function previewNavNext() {
    const list = previewableRows.value
    const i = list.findIndex((r) => r.path === previewItem.value?.path)
    if (i < 0 || i >= list.length - 1) return
    openPreview(list[i + 1])
  }

  return {
    previewNavMeta,
    previewNavPrev,
    previewNavNext,
  }
}

