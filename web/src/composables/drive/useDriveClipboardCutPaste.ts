import { ref } from 'vue'
import { driveUserRename } from '../../api/driveUserClient'

export function useDriveClipboardCutPaste(opts: {
  getCurrentDir: () => string
  onMoved?: (name: string) => void | Promise<void>
}) {
  const clipboardCutPath = ref<string>('')

  function cut(path: string) {
    clipboardCutPath.value = path
  }

  async function pasteCut() {
    const from = clipboardCutPath.value
    if (!from) return
    const name = from.includes('/') ? from.split('/').pop() || '' : from
    const base = opts.getCurrentDir() ? opts.getCurrentDir().replace(/\/+$/, '') : ''
    const to = base ? `${base}/${name}` : name

    await driveUserRename(from, to)
    clipboardCutPath.value = ''
    await opts.onMoved?.(name)
  }

  return { clipboardCutPath, cut, pasteCut }
}

