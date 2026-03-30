import { ref, type Ref } from 'vue'

type UseDriveDropZoneOptions = {
  canUpload: Ref<boolean>
  enqueueFiles: (files: File[]) => void
}

export function useDriveDropZone(options: UseDriveDropZoneOptions) {
  const dragOver = ref(false)
  let dragDepth = 0

  function onDragEnter() {
    dragDepth += 1
    dragOver.value = true
  }

  function onDragLeave() {
    dragDepth -= 1
    if (dragDepth <= 0) {
      dragDepth = 0
      dragOver.value = false
    }
  }

  function onDragOver() {
    dragOver.value = true
  }

  function onDropFiles(event: DragEvent) {
    dragDepth = 0
    dragOver.value = false
    if (!options.canUpload.value) return
    const transfer = event.dataTransfer
    if (!transfer?.files?.length) return
    options.enqueueFiles(Array.from(transfer.files))
  }

  return {
    dragOver,
    onDragEnter,
    onDragLeave,
    onDragOver,
    onDropFiles,
  }
}

