import { ref } from 'vue'
import type { DriveEntry } from '../../types/drive'
import { driveUserMkdir, driveUserRename } from '../../api/driveUserClient'

export function useDriveModals(opts: {
  getCurrentDir: () => string
  onChanged?: () => void | Promise<void>
}) {
  const renameModal = ref(false)
  const renaming = ref(false)
  const renameItem = ref<DriveEntry | null>(null)
  const renameValue = ref('')

  const mkdirModal = ref(false)
  const mkdirValue = ref('')
  const mkdiring = ref(false)

  function openRename(item: DriveEntry) {
    renameItem.value = item
    renameValue.value = item.name
    renameModal.value = true
  }

  async function confirmRename() {
    const item = renameItem.value
    const nextName = renameValue.value.trim()
    if (!item || !nextName) return
    const parent = item.path.includes('/') ? item.path.split('/').slice(0, -1).join('/') : ''
    const to = parent ? `${parent}/${nextName}` : nextName

    renaming.value = true
    try {
      await driveUserRename(item.path, to)
      renameModal.value = false
      await opts.onChanged?.()
    } finally {
      renaming.value = false
    }
  }

  function openMkdir() {
    mkdirValue.value = ''
    mkdirModal.value = true
  }

  async function confirmMkdir() {
    const name = mkdirValue.value.trim()
    if (!name) return
    const base = opts.getCurrentDir() ? opts.getCurrentDir().replace(/\/+$/, '') : ''
    const target = base ? `${base}/${name}` : name

    mkdiring.value = true
    try {
      await driveUserMkdir(target)
      mkdirModal.value = false
      await opts.onChanged?.()
    } finally {
      mkdiring.value = false
    }
  }

  return {
    renameModal,
    renaming,
    renameItem,
    renameValue,
    openRename,
    confirmRename,
    mkdirModal,
    mkdirValue,
    mkdiring,
    openMkdir,
    confirmMkdir,
  }
}

