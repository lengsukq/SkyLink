import { reactive, ref, type Ref } from 'vue'
import api from '../../api/client'
import { notifyError, notifySuccess } from '../../ui/notify'
import { getApiErrorMessage } from '../../utils/apiError'
import type {
  EasyTierDaemonActionResponse,
  EasyTierDaemonLogsResponse,
  EasyTierDaemonStatusResponse,
  EasyTierReleasePortResponse,
} from './types'

type UseEasyTierDaemonOptions = {
  profilePath: (path: string) => string
  selectedVersion: Ref<string | null>
  imageTag: Ref<string>
  loadStatus: () => Promise<void>
}

export function useEasyTierDaemon(options: UseEasyTierDaemonOptions) {
  function getHint(error: unknown): string {
    if (error == null || typeof error !== 'object') return ''
    const response = (error as { response?: { data?: unknown } }).response
    const data = response?.data
    if (data == null || typeof data !== 'object') return ''
    const hint = (data as { hint?: unknown }).hint
    return typeof hint === 'string' ? hint : ''
  }

  const daemonStatus = reactive({
    running: false,
    pid: 0,
    started_version: '',
  })
  const daemonModeEnabled = ref(false)
  const daemonLogs = ref('')
  const daemonLogsLoading = ref(false)
  const releasePortLoading = ref(false)

  async function loadDaemonStatus() {
    try {
      const { data } = await api.get<EasyTierDaemonStatusResponse>(options.profilePath('/daemon/status'))
      daemonStatus.running = !!data?.running
      daemonStatus.pid = data?.pid || 0
      daemonStatus.started_version = data?.started_version || ''
      daemonModeEnabled.value = !!data?.daemon_mode_enabled
    } catch (_) {
      daemonStatus.running = false
      daemonStatus.pid = 0
      daemonStatus.started_version = ''
      daemonModeEnabled.value = false
    }
  }

  async function loadDaemonLogs() {
    if (!daemonModeEnabled.value) return
    daemonLogsLoading.value = true
    try {
      const { data } = await api.get<EasyTierDaemonLogsResponse>(options.profilePath('/daemon/logs'))
      daemonLogs.value = data?.logs ?? ''
    } catch (_) {
      daemonLogs.value = ''
    } finally {
      daemonLogsLoading.value = false
    }
  }

  async function startDaemon() {
    const imageTag = options.selectedVersion.value || options.imageTag.value || ''
    try {
      const { data } = await api.post<EasyTierDaemonActionResponse>(options.profilePath('/daemon/start'), { image_tag: imageTag || undefined })
      notifySuccess('已启动', data?.message || 'EasyTier daemon 已启动。')
      await loadDaemonStatus()
      await options.loadStatus()
      await loadDaemonLogs()
    } catch (e: unknown) {
      const msg = getApiErrorMessage(e, '启动失败')
      const hint = getHint(e)
      notifyError(msg, hint || undefined)
      await loadDaemonStatus()
    }
  }

  async function stopDaemon() {
    try {
      const { data } = await api.post<EasyTierDaemonActionResponse>(options.profilePath('/daemon/stop'))
      notifySuccess('已停止', data?.message || 'EasyTier daemon 已停止。')
      await loadDaemonStatus()
      await options.loadStatus()
      await loadDaemonLogs()
    } catch (_) {}
  }

  async function restartDaemon() {
    const imageTag = options.selectedVersion.value || options.imageTag.value || ''
    try {
      const { data } = await api.post<EasyTierDaemonActionResponse>(options.profilePath('/daemon/restart'), { image_tag: imageTag || undefined })
      notifySuccess('已重启', data?.message || 'EasyTier daemon 已重启。')
      await loadDaemonStatus()
      await options.loadStatus()
      await loadDaemonLogs()
    } catch (e: unknown) {
      const msg = getApiErrorMessage(e, '重启失败')
      const hint = getHint(e)
      notifyError(msg, hint || undefined)
      await loadDaemonStatus()
    }
  }

  function startOrRestartDaemon() {
    if (daemonStatus.running) {
      restartDaemon()
    } else {
      startDaemon()
    }
  }

  async function releasePort() {
    releasePortLoading.value = true
    try {
      const { data } = await api.post<EasyTierReleasePortResponse>(options.profilePath('/daemon/release-port'))
      const msg = data?.message || '已解除端口占用。'
      if (data?.killed) {
        const portsStr = (data.ports_freed || []).length ? ` 端口 ${(data.ports_freed || []).join(', ')}` : ''
        notifySuccess('解除端口占用', `${msg} 已结束 ${data.killed} 个进程${portsStr}。`)
      } else {
        notifySuccess('解除端口占用', msg)
      }
      await loadDaemonStatus()
    } catch (e: unknown) {
      notifyError(getApiErrorMessage(e, '解除端口占用失败'))
    } finally {
      releasePortLoading.value = false
    }
  }

  return {
    daemonStatus,
    daemonModeEnabled,
    daemonLogs,
    daemonLogsLoading,
    releasePortLoading,
    loadDaemonStatus,
    loadDaemonLogs,
    startOrRestartDaemon,
    stopDaemon,
    releasePort,
  }
}
