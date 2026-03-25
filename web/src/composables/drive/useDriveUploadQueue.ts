import { ref } from 'vue'
import { DRIVE_UPLOAD_MAX_CONCURRENCY } from '../../constants/drive'
import { driveUserUpload } from '../../api/driveUserClient'

export type DriveUploadQueueItem = {
  id: string
  name: string
  file: File
  percent: number
  status: 'queued' | 'uploading' | 'done' | 'error'
}

export function useDriveUploadQueue(opts: {
  getTargetDir: () => string
  onSettled?: () => void | Promise<void>
}) {
  const queue = ref<DriveUploadQueueItem[]>([])
  const uploading = ref(0)

  function enqueueFiles(files: File[]) {
    for (const f of files) {
      queue.value.push({
        id: `${Date.now()}-${Math.random()}`,
        name: f.name,
        file: f,
        percent: 0,
        status: 'queued',
      })
    }
    pump()
  }

  function pump() {
    if (uploading.value >= DRIVE_UPLOAD_MAX_CONCURRENCY) return
    const next = queue.value.find((x) => x.status === 'queued')
    if (!next) return

    uploading.value += 1
    next.status = 'uploading'
    driveUserUpload(opts.getTargetDir(), next.file, {
      onProgress: (loaded, total) => {
        if (total && total > 0) next.percent = Math.min(100, Math.round((loaded / total) * 100))
      },
    })
      .then(() => {
        next.status = 'done'
        next.percent = 100
      })
      .catch(() => {
        next.status = 'error'
      })
      .finally(async () => {
        uploading.value -= 1
        pump()
        await opts.onSettled?.()
      })
  }

  return { queue, uploading, enqueueFiles }
}

