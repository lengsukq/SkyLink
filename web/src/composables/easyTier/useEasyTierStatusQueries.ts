import { computed, reactive, ref, type Ref } from 'vue'
import api from '../../api/client'
import { notifySuccess } from '../../ui/notify'
import { parseCliPeerTable, getCliPeerSummary } from './cliRawParser'
import type { EasyTierCliOutputResponse, EasyTierStatusResponse, EasyTierStatusState, EasyTierVersionCheckResponse } from './types'

type UseEasyTierStatusQueriesOptions = {
  profilePath: (path: string) => string
  imageTag: Ref<string>
}

export function useEasyTierStatusQueries(options: UseEasyTierStatusQueriesOptions) {
  const statusLoading = ref(false)
  const lastStatusUpdated = ref('')
  const status = reactive<EasyTierStatusState>({
    ok: false,
    error: '',
    hint: '',
    version: '',
    self_ipv4: '',
    self_hostname: '',
    peers: [],
    routes: [],
  })

  const versionCheckLoading = ref(false)
  const versionCheck = reactive({
    current_version: '',
    latest_version: '',
    update_available: false,
    release_url: '',
  })

  const cliRawTarget = ref<'peer' | 'route' | 'node' | 'version'>('peer')
  const cliRawStdout = ref('')
  const cliRawStderr = ref('')
  const cliRawMetaError = ref('')
  const cliRawLoading = ref(false)

  const parsedCliPeerRows = computed(() => {
    if (cliRawTarget.value !== 'peer') return []
    if (!cliRawStdout.value) return []
    return parseCliPeerTable(cliRawStdout.value)
  })
  const cliPeerSummary = computed(() => getCliPeerSummary(parsedCliPeerRows.value))

  async function loadStatus() {
    statusLoading.value = true
    status.hint = ''
    try {
      const { data } = await api.get<EasyTierStatusResponse>(options.profilePath('/status'))
      status.ok = data?.ok ?? false
      status.error = data?.error || ''
      status.hint = data?.hint || ''
      status.version = data?.version || ''
      status.self_ipv4 = data?.self_ipv4 || ''
      status.self_hostname = data?.self_hostname || ''
      status.peers = data?.peers || []
      status.routes = data?.routes || []
      lastStatusUpdated.value = new Date().toLocaleTimeString()
    } catch (_) {
      status.error = '无法获取 EasyTier 状态，可能未启用或 RPC 地址不可达。'
      status.hint = '请确认 EasyTier 守护进程已运行、RPC 地址正确；CLI 应与 easytier-core 同目录（下载完整包）或已在 PATH 中。'
      status.peers = []
      status.routes = []
      lastStatusUpdated.value = new Date().toLocaleTimeString()
    } finally {
      statusLoading.value = false
    }
  }

  async function checkUpdate() {
    versionCheckLoading.value = true
    try {
      const { data } = await api.get<EasyTierVersionCheckResponse>('/easytier/version/check')
      versionCheck.current_version = data?.current_version || ''
      versionCheck.latest_version = data?.latest_version || ''
      versionCheck.update_available = !!data?.update_available
      versionCheck.release_url = data?.release_url || ''
    } catch (_) {
      // no-op
    } finally {
      versionCheckLoading.value = false
    }
  }

  function useLatestVersion() {
    if (versionCheck.latest_version) {
      options.imageTag.value = versionCheck.latest_version
      notifySuccess('已填入', '请保存配置并重启 EasyTier 守护进程以完成升级。')
    }
  }

  async function loadCliRaw() {
    cliRawLoading.value = true
    cliRawMetaError.value = ''
    cliRawStdout.value = ''
    cliRawStderr.value = ''
    try {
      const { data } = await api.get<EasyTierCliOutputResponse>(options.profilePath('/cli-output'), {
        params: { target: cliRawTarget.value },
        silentError: true,
      })
      if (data?.hint && data?.ok === false) {
        cliRawMetaError.value = data.hint
        return
      }
      cliRawStdout.value = typeof data?.stdout === 'string' ? data.stdout : ''
      cliRawStderr.value = typeof data?.stderr === 'string' ? data.stderr : ''
      if (data?.error) {
        cliRawMetaError.value = String(data.error)
      }
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } }; message?: string }
      cliRawMetaError.value = err?.response?.data?.error || err?.message || String(e)
    } finally {
      cliRawLoading.value = false
    }
  }

  return {
    statusLoading,
    lastStatusUpdated,
    status,
    versionCheckLoading,
    versionCheck,
    checkUpdate,
    useLatestVersion,
    cliRawTarget,
    cliRawStdout,
    cliRawStderr,
    cliRawMetaError,
    cliRawLoading,
    loadCliRaw,
    parsedCliPeerRows,
    cliPeerSummary,
    loadStatus,
  }
}
