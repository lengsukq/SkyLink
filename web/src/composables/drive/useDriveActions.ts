import { computed, ref } from 'vue'
import type { DriveEntry, PreviewKind } from '../../types/drive'
import { driveUserDelete, driveUserDownloadBlob } from '../../api/driveUserClient'
import { copyToClipboard } from '../../utils/clipboard'
import { previewKindForEntry } from '../../utils/drivePreview'

export function useDriveActions(opts: {
  refresh: (reset: boolean) => Promise<void>
  notifySuccess: (title: string, content: string) => void
  notifyError: (title: string, content?: string) => void
}) {
  const previewOpen = ref(false)
  const previewItem = ref<DriveEntry | null>(null)

  const previewKind = computed<PreviewKind>(() => {
    const r = previewItem.value
    if (!r) return 'unknown'
    return previewKindForEntry(r)
  })

  function openPreview(row: DriveEntry) {
    if (row.is_dir) return
    previewItem.value = row
    previewOpen.value = true
  }

  function openOrEnter(item: DriveEntry, goToDir: (p: string) => void) {
    if (item.is_dir) {
      goToDir(item.path)
      return
    }
    openPreview(item)
  }

  async function download(row: DriveEntry) {
    try {
      const blob = await driveUserDownloadBlob(row.path)
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = row.name
      a.click()
      URL.revokeObjectURL(url)
    } catch (e: any) {
      opts.notifyError('下载失败', e?.response?.data?.error || e?.message || String(e))
    }
  }

  async function remove(row: DriveEntry) {
    try {
      await driveUserDelete(row.path)
      opts.notifySuccess('已删除', row.name)
      await opts.refresh(true)
    } catch (e: any) {
      opts.notifyError('删除失败', e?.response?.data?.error || e?.message || String(e))
    }
  }

  async function batchDelete(paths: string[]) {
    const targets = [...paths]
    if (!targets.length) return
    try {
      for (const p of targets) {
        await driveUserDelete(p)
      }
      opts.notifySuccess('已删除', `共 ${targets.length} 项`)
      await opts.refresh(true)
    } catch (e: any) {
      opts.notifyError('批量删除失败', e?.response?.data?.error || e?.message || String(e))
    }
  }

  async function copyPath(path: string) {
    try {
      await copyToClipboard(path)
      opts.notifySuccess('已复制', '路径已复制到剪贴板')
    } catch (e: any) {
      opts.notifyError('复制失败', e?.message || String(e))
    }
  }

  return {
    previewOpen,
    previewItem,
    previewKind,
    openPreview,
    openOrEnter,
    download,
    remove,
    batchDelete,
    copyPath,
  }
}

