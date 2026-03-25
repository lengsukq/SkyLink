import { computed, ref } from 'vue'
import type { DriveEntry } from '../../types/drive'

export type DriveContextActionKey = 'preview' | 'download' | 'copy-path' | 'rename' | 'cut' | 'delete'

export function useDriveContextMenu(opts: {
  onAction: (key: DriveContextActionKey, item: DriveEntry) => void | Promise<void>
  onSelectItem?: (item: DriveEntry) => void
}) {
  const ctx = ref<{ show: boolean; x: number; y: number; item: DriveEntry | null }>({
    show: false,
    x: 0,
    y: 0,
    item: null,
  })

  const options = computed(() => {
    const item = ctx.value.item
    const isFile = !!item && !item.is_dir
    return [
      { label: '预览', key: 'preview', disabled: !isFile },
      { label: '下载', key: 'download', disabled: !isFile },
      { label: '复制路径', key: 'copy-path', disabled: !item },
      { label: '重命名', key: 'rename', disabled: !item },
      { label: '剪切', key: 'cut', disabled: !item },
      { label: '删除', key: 'delete', disabled: !item },
    ]
  })

  function openAt(item: DriveEntry, e: MouseEvent) {
    ctx.value.item = item
    ctx.value.x = e.clientX
    ctx.value.y = e.clientY
    ctx.value.show = true
    opts.onSelectItem?.(item)
  }

  function close() {
    ctx.value.show = false
  }

  async function onSelect(key: string) {
    const item = ctx.value.item
    close()
    if (!item) return
    await opts.onAction(key as DriveContextActionKey, item)
  }

  return { ctx, options, openAt, close, onSelect }
}

