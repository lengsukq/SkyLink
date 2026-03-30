import { ref } from 'vue'
import { driveUserIndexRebuild, driveUserIndexStatus } from '../../api/driveUserClient'
import { getApiErrorMessage } from '../../utils/apiError'

type UseDriveIndexSyncOptions = {
  refreshWithToast: (reset: boolean) => Promise<void>
  notifySuccess: (title: string, message: string) => void
  notifyError: (title: string, message: string) => void
}

const INDEX_POLL_MAX = 7200
const INDEX_POLL_MS = 500

export function useDriveIndexSync(options: UseDriveIndexSyncOptions) {
  const reindexing = ref(false)

  async function syncLocalIndex() {
    reindexing.value = true
    try {
      try {
        await driveUserIndexRebuild()
      } catch (e: any) {
        const msg = String(e?.response?.data?.error ?? '')
        if (!msg.includes('already running')) throw e
      }
      let timedOut = false
      for (let i = 0; i < INDEX_POLL_MAX; i++) {
        const { status } = await driveUserIndexStatus()
        if (!status) break
        if (!status.running) {
          if (status.last_error) {
            options.notifyError('同步失败', status.last_error)
            return
          }
          break
        }
        if (i === INDEX_POLL_MAX - 1) {
          timedOut = true
          break
        }
        await new Promise((r) => setTimeout(r, INDEX_POLL_MS))
      }
      if (timedOut) {
        options.notifySuccess('提示', '同步任务仍在进行，请稍后点击「查询」刷新列表')
      } else {
        options.notifySuccess('同步完成', '列表已与本地目录对齐')
      }
      await options.refreshWithToast(true)
    } catch (e: unknown) {
      options.notifyError('同步失败', getApiErrorMessage(e))
    } finally {
      reindexing.value = false
    }
  }

  return {
    reindexing,
    syncLocalIndex,
  }
}
