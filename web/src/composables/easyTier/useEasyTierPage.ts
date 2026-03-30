/**
 * EasyTier 状态与操作（由 Windows 工具页中的 EasyTierPanel 使用）。
 */
import { ref, reactive, onMounted, onUnmounted, computed } from 'vue'
import api from '../../api/client'
import { useEasyTierPeerResident } from './useEasyTierPeerResident'
import { cliPeerColumns, nodeTableColumns, routeColumns } from './easyTierTableColumns'
import { useEasyTierRuntime } from './useEasyTierRuntime'
import { useEasyTierConfigForm } from './useEasyTierConfigForm'
import { useEasyTierDaemon } from './useEasyTierDaemon'
import { useEasyTierProfiles } from './useEasyTierProfiles'
import { useEasyTierStatusQueries } from './useEasyTierStatusQueries'
import type {
  DisplayNodeRow,
  EasyTierFormState,
  EasyTierConfigResponse,
  EasyTierSettingsResponse,
} from './types'
export function useEasyTierPage() {
const formRef = ref(null)
const DEFAULT_PUBLIC_SERVER = 'tcp://public.easytier.cn:11010'
const form = reactive<EasyTierFormState>({
  network_name: '',
  network_secret: '',
  peers: '',
  ipv4: '',
  enabled: false,
  image_tag: '',
  rpc_portal: '',
  hostname: '',
  external_node: '',
  proxy_networks: '',
  dhcp: false,
})
const networkMode = ref('manual')
const publicServer = ref(DEFAULT_PUBLIC_SERVER)
const showAdvanced = ref(false)
const saving = ref(false)
/** 后端显式为 true 时集成可用（仅 Windows SkyLink）。 */
const easytierHostSupported = ref(false)
const easytierAutostart = ref(false)
const easytierAutostartSaving = ref(false)

const parsedPeers = computed(() =>
  form.peers
    .split(/[\n,]+/)
    .map((s) => s.trim())
    .filter(Boolean)
)
const peerCount = computed(() => parsedPeers.value.length)
const hasNodeConfig = computed(() => {
  if (networkMode.value === 'standalone') return true
  return parsedPeers.value.length > 0
})
const shouldPromptNodeConfig = computed(() => easytierHostSupported.value && !hasNodeConfig.value)
const runtimeOpsDisabledReason = computed(() => {
  if (!easytierHostSupported.value) return '仅在 Windows 上可用'
  if (!daemonModeEnabled.value) return '需在配置中开启 easytier.daemon_enabled 并重启 SkyLink'
  if (!hasNodeConfig.value) return '请先配置节点信息'
  if (!form.enabled) return '请先在配置中开启“启用”并保存'
  return ''
})
const runtimeOpsDisabled = computed(() => runtimeOpsDisabledReason.value !== '')

const {
  platform,
  releasesList,
  releasesError,
  platformsList,
  currentPlatformLabel,
  releaseOptions,
  platformOptions,
  selectedVersion,
  selectedPlatformKey,
  runtimeInstalling,
  runtimeRemoving,
  runtimeInstalled,
  runtimeVersion,
  platformError,
  runtimeError,
  installedList,
  removingInstalledKey,
  installedOptionsForStatus,
  selectedInstalledKey,
  installedListColumns,
  loadPlatform,
  loadReleases,
  loadPlatforms,
  loadRuntimeInstalled,
  loadInstalledList,
  onVersionOrPlatformChange,
  installRuntime,
  removeRuntime,
  ensureRuntimeDefaults,
} = useEasyTierRuntime({
  form,
  easytierHostSupported,
})

const aggregatedErrors = computed(() => {
  const list: string[] = []
  if (status.error) list.push(status.error)
  if (releasesError.value) list.push(releasesError.value)
  if (runtimeError.value) list.push(runtimeError.value)
  if (platformError.value) list.push(platformError.value)
  return list
})

// 展示用节点列表：本机行 + peers，用于「节点信息」表
const displayNodes = computed(() => {
  const list: DisplayNodeRow[] = []
  if (status.self_ipv4 || status.self_hostname) {
    list.push({
      ipv4: status.self_ipv4 || '—',
      hostname: status.self_hostname || '—',
      route: '本机',
      tunnel: '—',
      latency_ms: null,
      version: status.version || '—',
    })
  }
  ;(status.peers || []).forEach((p) => {
    list.push({
      ipv4: p.ipv4 || '—',
      hostname: p.hostname || '—',
      route: 'p2p',
      tunnel: p.tunnel || '—',
      latency_ms: p.latency_ms ?? null,
      version: p.version || '—',
    })
  })
  return list
})

function onDHCPChange(val: boolean) {
  form.dhcp = val
  if (val) {
    form.ipv4 = ''
  }
}

function onIPv4Change(val: string) {
  form.ipv4 = val
  if (val && form.dhcp) {
    form.dhcp = false
  }
}

async function loadConfig() {
  try {
    const { data } = await api.get<EasyTierConfigResponse>('/easytier/config')
    if (data) {
      form.network_name = data.network_name || ''
      form.network_secret = data.network_secret || ''
      form.peers = data.peers || ''
      form.ipv4 = data.ipv4 || ''
      form.enabled = !!data.enabled
      form.image_tag = data.image_tag || ''
      form.rpc_portal = data.rpc_portal || ''
      form.hostname = data.hostname || ''
      form.external_node = data.external_node || ''
      form.proxy_networks = data.proxy_networks || ''
      form.dhcp = !!data.dhcp
      const peers = (data.peers || '').trim()
      form.peers = peers
      if (!peers) {
        networkMode.value = 'standalone'
      } else if (peers.includes('public.easytier.cn')) {
        networkMode.value = 'public'
        const first = peers
          .split(/[\n,]+/)
          .map((s: string) => s.trim())
          .find(Boolean)
        publicServer.value = first || DEFAULT_PUBLIC_SERVER
      } else {
        networkMode.value = 'manual'
      }
      if (!publicServer.value) {
        publicServer.value = DEFAULT_PUBLIC_SERVER
      }
    }
  } catch (_) {}
  await loadDaemonStatus()
}

const {
  profiles,
  activeProfileId,
  newProfileName,
  profileOptions,
  canDeleteProfile,
  profilePath,
  loadProfiles,
  onProfileChange,
  createProfile,
  deleteCurrentProfile,
} = useEasyTierProfiles({
  imageTag: computed(() => form.image_tag),
  loadConfig,
  loadStatus,
  loadDaemonLogs: async () => {
    await loadDaemonLogs()
  },
})

const {
  peerResidentRows,
  peerResidentColumns,
  loadPeerResidentOnly,
  startStatusPolling,
  stopStatusPolling,
} = useEasyTierPeerResident({
  easytierHostSupported,
  profilePath,
})

const {
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
} = useEasyTierStatusQueries({
  profilePath,
  imageTag: computed(() => form.image_tag),
})

const {
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
} = useEasyTierDaemon({
  profilePath,
  selectedVersion,
  imageTag: computed(() => form.image_tag),
  loadStatus,
})

const { save, saveAndRestart, saveByModal, saveAndRestartByModal } = useEasyTierConfigForm({
  form,
  networkMode,
  parsedPeers,
  publicServer,
  saving,
  profilePath,
  loadConfig,
  loadStatus,
  loadDaemonStatus,
})

async function loadEasyTierAutostart() {
  try {
    const { data } = await api.get<EasyTierSettingsResponse>('/easytier/settings')
    easytierAutostart.value = !!data?.autostart_on_startup
  } catch (_) {
    easytierAutostart.value = false
  }
}


async function onToggleEasyTierAutostart(val: boolean) {
  const previous = easytierAutostart.value
  easytierAutostart.value = val
  easytierAutostartSaving.value = true
  try {
    await api.put('/easytier/settings', { autostart_on_startup: !!val })
    notifySuccess('已更新', 'SkyLink 启动时自动启动 EasyTier 已更新')
  } catch (_) {
    easytierAutostart.value = previous
  } finally {
    easytierAutostartSaving.value = false
  }
}

onMounted(async () => {
  await loadProfiles()
  await loadPlatform()
  await loadEasyTierAutostart()
  await loadConfig()
  loadStatus()
  await loadReleases()
  await loadPlatforms()
  ensureRuntimeDefaults()
  await loadRuntimeInstalled()
  await loadInstalledList()
  await loadPeerResidentOnly()
  startStatusPolling()
})
onUnmounted(() => {
  stopStatusPolling()
})

  const profileGroup = {
    profiles,
    activeProfileId,
    newProfileName,
    profileOptions,
    canDeleteProfile,
    loadProfiles,
    onProfileChange,
    createProfile,
    deleteCurrentProfile,
  }

  const metaGroup = {
    easytierHostSupported,
  }

  const configGroup = {
    formRef,
    form,
    networkMode,
    publicServer,
    showAdvanced,
    saving,
    parsedPeers,
    peerCount,
    hasNodeConfig,
    shouldPromptNodeConfig,
    onDHCPChange,
    onIPv4Change,
    loadConfig,
    save,
    saveAndRestart,
    saveByModal,
    saveAndRestartByModal,
  }

  const statusGroup = {
    statusLoading,
    lastStatusUpdated,
    status,
    aggregatedErrors,
    hasNodeConfig,
    runtimeOpsDisabled,
    runtimeOpsDisabledReason,
    displayNodes,
    nodeTableColumns,
    routeColumns,
    loadStatus,
    cliRawTarget,
    cliRawStdout,
    cliRawStderr,
    cliRawMetaError,
    cliRawLoading,
    loadCliRaw,
    parsedCliPeerRows,
    cliPeerSummary,
    cliPeerColumns,
    peerResidentRows,
    peerResidentColumns,
  }

  const runtimeGroup = {
    platform,
    releasesList,
    releasesError,
    platformsList,
    currentPlatformLabel,
    releaseOptions,
    platformOptions,
    selectedVersion,
    selectedPlatformKey,
    runtimeInstalling,
    runtimeRemoving,
    runtimeInstalled,
    runtimeVersion,
    platformError,
    runtimeError,
    installedList,
    removingInstalledKey,
    installedOptionsForStatus,
    selectedInstalledKey,
    installedListColumns,
    loadPlatform,
    loadReleases,
    loadPlatforms,
    loadRuntimeInstalled,
    loadInstalledList,
    onVersionOrPlatformChange,
    installRuntime,
    removeRuntime,
    versionCheckLoading,
    versionCheck,
    checkUpdate,
    useLatestVersion,
  }

  const daemonGroup = {
    daemonStatus,
    daemonModeEnabled,
    daemonLogs,
    daemonLogsLoading,
    releasePortLoading,
    loadDaemonStatus,
    loadDaemonLogs,
    stopDaemon,
    startOrRestartDaemon,
    releasePort,
    easytierAutostart,
    easytierAutostartSaving,
    onToggleEasyTierAutostart,
  }

  return {
    meta: metaGroup,
    profile: profileGroup,
    config: configGroup,
    statusView: statusGroup,
    runtime: runtimeGroup,
    daemon: daemonGroup,
  }
}
