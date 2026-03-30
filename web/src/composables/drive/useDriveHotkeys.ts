import { onMounted, onUnmounted, type Ref } from 'vue'
import type { DriveEntry } from '../../types/drive'

type UseDriveHotkeysOptions = {
  previewOpen: Ref<boolean>
  selectedItem: Ref<DriveEntry | null>
  selectedPaths: Ref<string[]>
  recursive: Ref<boolean>
  path: Ref<string>
  openOrEnter: (item: DriveEntry) => void
  goTo: (path: string) => void
  confirmBatchDelete: () => void
}

export function useDriveHotkeys(options: UseDriveHotkeysOptions) {
  function onGlobalKeydown(e: KeyboardEvent) {
    if (isTypingTarget(e.target)) return

    if (e.key === 'Escape' && options.previewOpen.value) {
      options.previewOpen.value = false
      e.preventDefault()
      return
    }

    if (e.key === 'Enter' && options.selectedItem.value) {
      options.openOrEnter(options.selectedItem.value)
      e.preventDefault()
      return
    }

    if (e.key === 'Backspace' && !options.recursive.value) {
      const currentPath = (options.path.value || '').trim()
      if (currentPath) {
        e.preventDefault()
        options.goTo(parentDirPath(currentPath))
      }
      return
    }

    if (e.key === 'Delete' && options.selectedPaths.value.length) {
      e.preventDefault()
      options.confirmBatchDelete()
    }
  }

  onMounted(() => window.addEventListener('keydown', onGlobalKeydown))
  onUnmounted(() => window.removeEventListener('keydown', onGlobalKeydown))
}

function parentDirPath(path: string): string {
  const normalized = path.replace(/\\/g, '/').replace(/\/+$/, '')
  const slashIndex = normalized.lastIndexOf('/')
  return slashIndex <= 0 ? '' : normalized.slice(0, slashIndex)
}

function isTypingTarget(target: EventTarget | null): boolean {
  if (!target || !(target instanceof HTMLElement)) return false
  const tag = target.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true
  return target.isContentEditable
}

