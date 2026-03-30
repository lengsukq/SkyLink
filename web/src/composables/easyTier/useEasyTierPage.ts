/**
 * EasyTier 状态与操作（由 Windows 工具页中的 EasyTierPanel 使用）。
 */
import { ref, reactive, onMounted, h, computed } from 'vue'
import { NButton } from 'naive-ui'
import api from '../../api/client'
import { notifySuccess, notifyError } from '../../ui/notify'
export function useEasyTierPage() {
const formRef = ref(null)
const DEFAULT_PUBLIC_SERVER = 'tcp://public.easytier.cn:11010'
const form = reactive({
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
const daemonStatus = reactive({
  running: false,
  pid: 0,
  last_start_error: '',
  started_version: '',
})
const daemonModeEnabled = ref(false)
const networkMode = ref('manual')
const publicServer = ref(DEFAULT_PUBLIC_SERVER)
const showAdvanced = ref(false)
const saving = ref(false)
const statusLoading = ref(false)
const versionCheckLoading = ref(false)
const lastStatusUpdated = ref('')

const status = reactive({
  ok: false,
  error: '',
  hint: '',
  version: '',
  self_ipv4: '',
  self_hostname: '',
  peers: [] as any[],
  routes: [] as any[],
})
const versionCheck = reactive({
  current_version: '',
  latest_version: '',
  update_available: false,
  release_url: '',
})
const platform = reactive({
  os: '',
  arch: '',
  label: '',
})
const releasesList = ref<any[]>([])
const releasesError = ref('')
const platformsList = ref<any[]>([])
const currentPlatformLabel = ref('')
/** 后端显式为 true 时集成可用（仅 Windows SkyLink）。 */
const easytierHostSupported = ref(false)
const releaseOptions = ref<any[]>([])
const platformOptions = ref<any[]>([])
const selectedVersion = ref<string | null>(null)
const selectedPlatformKey = ref<string | null>(null)
const runtimeInstalling = ref(false)
const runtimeRemoving = ref(false)
const runtimeInstalled = ref(false)
const runtimeVersion = ref('')
const platformError = ref('')
const runtimeError = ref('')
const installedList = ref<any[]>([])
const removingInstalledKey = ref('')
const daemonLogs = ref('')
const daemonLogsLoading = ref(false)
const releasePortLoading = ref(false)
const easytierAutostart = ref(false)
const easytierAutostartSaving = ref(false)
const cliRawTarget = ref<'peer' | 'route' | 'node' | 'version'>('peer')
const cliRawStdout = ref('')
const cliRawStderr = ref('')
const cliRawMetaError = ref('')
const cliRawLoading = ref(false)
const profiles = ref<any[]>([])
const activeProfileId = ref('')
const newProfileName = ref('')

const profileOptions = computed(() => profiles.value.map((p) => ({ label: p.name || p.id, value: p.id })))
const canDeleteProfile = computed(() => profiles.value.length > 1 && !!activeProfileId.value)

const aggregatedErrors = computed(() => {
  const list: string[] = []
  if (daemonStatus.last_start_error) list.push(`上次启动错误：${daemonStatus.last_start_error}`)
  if (status.error) list.push(status.error)
  if (releasesError.value) list.push(releasesError.value)
  if (runtimeError.value) list.push(runtimeError.value)
  if (platformError.value) list.push(platformError.value)
  return list
})

const installedOptionsForStatus = computed(() =>
  installedList.value.map((row) => ({
    label: `${row.version} (${row.os}/${row.arch})`,
    value: installedItemKey(row),
  }))
)

const selectedInstalledKey = computed({
  get() {
    const ver = selectedVersion.value || form.image_tag
    const plat = selectedPlatformKey.value
    if (!ver || !plat) return null
    const key = `${ver}-${plat.replace('/', '-')}`
    const found = installedList.value.some((row) => installedItemKey(row) === key)
    return found ? key : null
  },
  set(val) {
    if (!val) return
    const row = installedList.value.find((r) => installedItemKey(r) === val)
    if (row) {
      selectedVersion.value = row.version
      selectedPlatformKey.value = `${row.os}/${row.arch}`
    }
  },
})

// 展示用节点列表：本机行 + peers，用于「节点信息」表
const displayNodes = computed(() => {
  const list: any[] = []
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
      latency_ms: p.latency_ms,
      version: p.version || '—',
    })
  })
  return list
})

const nodeTableColumns = [
  { title: '虚拟IPv4地址', key: 'ipv4', width: 140, ellipsis: true },
  { title: '主机名', key: 'hostname', ellipsis: true },
  { title: '路由', key: 'route', width: 70 },
  { title: '协议', key: 'tunnel', width: 100, ellipsis: true },
  {
    title: '延迟',
    key: 'latency_ms',
    width: 80,
    render: (row: any) => (row.latency_ms != null && row.latency_ms !== '' ? `${Number(row.latency_ms)}ms` : '—'),
  },
  { title: '内核版本', key: 'version', width: 120, ellipsis: true },
]
const routeColumns = [
  { title: 'IPv4', key: 'ipv4', width: 120 },
  { title: 'Hostname', key: 'hostname', ellipsis: true },
  { title: 'Proxy CIDRs', key: 'proxy_cidrs', ellipsis: true },
  { title: 'NextHop', key: 'next_hop_ipv4', width: 120 },
]

function installedItemKey(row: any) {
  return `${row.version}-${row.os}-${row.arch}`
}

function isCurrentSelected(row: any) {
  const plat = selectedPlatformKey.value ? `${row.os}/${row.arch}` === selectedPlatformKey.value : false
  const ver = (selectedVersion.value || form.image_tag) && row.version === (selectedVersion.value || form.image_tag)
  return plat && ver
}

const installedListColumns = [
  { title: '版本', key: 'version', width: 100 },
  { title: '平台', key: 'platform', width: 110, render: (row: any) => `${row.os}/${row.arch}` },
  { title: '说明', key: 'current', width: 80, render: (row: any) => (isCurrentSelected(row) ? '当前使用' : '') },
  {
    title: '操作',
    key: 'action',
    width: 80,
    render: (row: any) => {
      const key = installedItemKey(row)
      return h(
        NButton,
        {
          size: 'tiny',
          tertiary: true,
          loading: removingInstalledKey.value === key,
          disabled:
            !easytierHostSupported.value ||
            (removingInstalledKey.value !== '' && removingInstalledKey.value !== key),
          onClick: () => removeInstalledItem(row),
        },
        { default: () => '移除' }
      )
    },
  },
]

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
    const { data } = await api.get('/easytier/config')
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

async function loadProfiles() {
  try {
    const { data } = await api.get('/easytier/profiles')
    profiles.value = data?.profiles || []
    activeProfileId.value = data?.active_profile_id || profiles.value[0]?.id || ''
  } catch (_) {
    profiles.value = []
    activeProfileId.value = ''
  }
}

function profilePath(path: string) {
  if (!activeProfileId.value) return `/easytier${path}`
  return `/easytier/profiles/${activeProfileId.value}${path}`
}

async function onProfileChange(profileId: string) {
  if (!profileId) return
  try {
    await api.put(`/easytier/profiles/active/${profileId}`)
    await loadConfig()
    await loadStatus()
    await loadDaemonLogs()
  } catch (e) {
    notifyError('切换实例失败', e?.response?.data?.error || e?.message || '未知错误')
  }
}

async function createProfile() {
  const name = (newProfileName.value || '').trim()
  if (!name) return
  try {
    const { data } = await api.post('/easytier/profiles', { name, config: { image_tag: form.image_tag || '' } })
    newProfileName.value = ''
    await loadProfiles()
    if (data?.id) {
      activeProfileId.value = data.id
      await onProfileChange(data.id)
    }
  } catch (e) {
    notifyError('新增实例失败', e?.response?.data?.error || e?.message || '未知错误')
  }
}

async function deleteCurrentProfile() {
  if (!canDeleteProfile.value) return
  try {
    await api.delete(`/easytier/profiles/${activeProfileId.value}`)
    await loadProfiles()
    if (activeProfileId.value) {
      await onProfileChange(activeProfileId.value)
    }
  } catch (e) {
    notifyError('删除实例失败', e?.response?.data?.error || e?.message || '未知错误')
  }
}

async function loadPlatform() {
  try {
    const { data } = await api.get('/easytier/platform')
    platform.os = data?.os || ''
    platform.arch = data?.arch || ''
    platform.label = data?.label || ''
    easytierHostSupported.value = data?.easytier_host_supported === true
    platformError.value = ''
  } catch (e) {
    platform.os = ''
    platform.arch = ''
    platform.label = ''
    easytierHostSupported.value = false
    platformError.value = '无法获取后端平台信息，运行时下载功能可能不可用。'
  }
}

async function loadReleases() {
  releasesError.value = ''
  try {
    const { data } = await api.get('/easytier/releases')
    const list = data?.releases || []
    releasesList.value = list
    releaseOptions.value = list.map((r: any) => ({ label: r.tag_name, value: r.tag_name }))
  } catch (_) {
    releasesList.value = []
    releaseOptions.value = []
    releasesError.value = '无法拉取 GitHub Releases 列表。'
  }
}

async function loadPlatforms() {
  try {
    const { data } = await api.get('/easytier/platforms')
    const list = data?.platforms || []
    platformsList.value = list
    currentPlatformLabel.value = data?.current?.label || ''
    if (data?.easytier_host_supported === true || data?.easytier_host_supported === false) {
      easytierHostSupported.value = data.easytier_host_supported === true
    }
    platformOptions.value = list.map((p: any) => ({
      label: p.label || `${p.os}/${p.arch}`,
      value: `${p.os}/${p.arch}`,
    }))
  } catch (_) {
    platformsList.value = []
    platformOptions.value = []
  }
}

async function loadRuntimeInstalled() {
  const version = selectedVersion.value || form.image_tag
  if (!version || !selectedPlatformKey.value) {
    runtimeInstalled.value = false
    return
  }
  const [osVal, arch] = selectedPlatformKey.value.split('/')
  if (!osVal || !arch) {
    runtimeInstalled.value = false
    return
  }
  try {
    const { data } = await api.get('/easytier/runtime/installed', {
      params: { version, os: osVal, arch },
    })
    runtimeInstalled.value = !!data?.installed
    if (data?.installed) {
      runtimeVersion.value = version
    }
  } catch (_) {
    runtimeInstalled.value = false
  }
}

async function loadInstalledList() {
  try {
    const { data } = await api.get('/easytier/runtime/list')
    installedList.value = data?.items || []
  } catch (_) {
    installedList.value = []
  }
}

async function removeInstalledItem(row: any) {
  const key = installedItemKey(row)
  removingInstalledKey.value = key
  try {
    await api.delete('/easytier/runtime', {
      data: { version: row.version, os: row.os, arch: row.arch },
    })
    notifySuccess('已移除', `${row.version} (${row.os}/${row.arch}) 已从本机删除。`)
    await loadInstalledList()
    await loadRuntimeInstalled()
  } catch (e) {
    const msg = e?.response?.data?.error || e?.message || '移除失败'
    notifyError('移除失败', msg)
  } finally {
    removingInstalledKey.value = ''
  }
}

function onVersionOrPlatformChange() {
  if (selectedVersion.value) {
    form.image_tag = selectedVersion.value
  }
  loadRuntimeInstalled()
}

async function save() {
  const errors = []
  if (form.enabled) {
    if (!form.network_name.trim()) {
      errors.push('请填写网络名（ET_NETWORK_NAME，对应 EasyTier network_identity.name）。')
    }
    if (!form.network_secret.trim()) {
      errors.push('请填写网络密钥（ET_NETWORK_SECRET，对应 EasyTier network_identity.secret）。')
    }
    if (networkMode.value !== 'standalone' && !form.peers.trim()) {
      errors.push('请至少填写一个初始节点（ET_PEERS，对应 peers[].uri）。')
    }
  }
  const ipv4 = form.ipv4.trim()
  if (ipv4) {
    const ipv4Pattern = /^(25[0-5]|2[0-4]\d|1?\d?\d)(\.(25[0-5]|2[0-4]\d|1?\d?\d)){3}$/
    if (!ipv4Pattern.test(ipv4)) {
      errors.push('本机 IPv4 格式不正确，请使用形如 10.144.144.1 的地址。')
    }
  }
  const cidrPattern = /^(25[0-5]|2[0-4]\d|1?\d?\d)(\.(25[0-5]|2[0-4]\d|1?\d?\d)){3}\/([0-9]|[12][0-9]|3[0-2])$/
  if (form.proxy_networks.trim()) {
    const items = form.proxy_networks
      .split(/[\n,]+/)
      .map((s) => s.trim())
      .filter(Boolean)
    const invalid = items.filter((item) => !cidrPattern.test(item))
    if (invalid.length) {
      errors.push(`子网代理（proxy-networks）格式不正确：${invalid.join('，')}。请使用形如 10.0.0.0/24 的 CIDR。`)
    }
  }
  if (errors.length) {
    notifyError('配置不完整或格式错误', errors.join('；'))
    return
  }
  if (networkMode.value === 'standalone') {
    form.peers = ''
  } else if (networkMode.value === 'public') {
    form.peers = (publicServer.value || '').trim()
  }
  saving.value = true
  try {
    const { data } = await api.put('/easytier/config', form)
    notifySuccess('已保存', '若需使网络/高级配置生效，请点击上方状态卡片中的「重启」。')
    await loadConfig()
  } finally {
    saving.value = false
  }
}

async function saveAndRestart() {
  const errors = []
  if (form.enabled) {
    if (!form.network_name.trim()) errors.push('请填写网络名。')
    if (!form.network_secret.trim()) errors.push('请填写网络密钥。')
    if (networkMode.value !== 'standalone' && !form.peers.trim()) errors.push('请至少填写一个初始节点。')
  }
  const ipv4 = form.ipv4.trim()
  if (ipv4) {
    const ipv4Pattern = /^(25[0-5]|2[0-4]\d|1?\d?\d)(\.(25[0-5]|2[0-4]\d|1?\d?\d)){3}$/
    if (!ipv4Pattern.test(ipv4)) errors.push('本机 IPv4 格式不正确。')
  }
  const cidrPattern = /^(25[0-5]|2[0-4]\d|1?\d?\d)(\.(25[0-5]|2[0-4]\d|1?\d?\d)){3}\/([0-9]|[12][0-9]|3[0-2])$/
  if (form.proxy_networks.trim()) {
    const items = form.proxy_networks.split(/[\n,]+/).map((s) => s.trim()).filter(Boolean)
    const invalid = items.filter((item) => !cidrPattern.test(item))
    if (invalid.length) errors.push(`子网代理格式不正确：${invalid.join('，')}`)
  }
  if (errors.length) {
    notifyError('配置不完整或格式错误', errors.join('；'))
    return
  }
  if (networkMode.value === 'standalone') form.peers = ''
  else if (networkMode.value === 'public') form.peers = (publicServer.value || '').trim()

  saving.value = true
  try {
    await api.put('/easytier/config', form)
    await loadConfig()
    try {
      const { data } = await api.post(profilePath('/daemon/restart'))
      notifySuccess('已保存并重启', data?.message || '配置已保存，EasyTier 守护进程已重启。')
      await loadDaemonStatus()
      await loadStatus()
    } catch (e) {
      const err = e?.response?.data
      const msg = err?.error || e?.message || '重启失败'
      const hint = err?.hint || ''
      notifyError('配置已保存，但重启失败', hint ? `${msg} ${hint}` : msg)
      await loadDaemonStatus()
    }
  } catch (e) {
    const msg = e?.response?.data?.error || e?.message || '保存失败'
    notifyError('保存失败', msg)
  } finally {
    saving.value = false
  }
}

async function loadStatus() {
  statusLoading.value = true
  status.hint = ''
  try {
    const { data } = await api.get(profilePath('/status'))
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
    status.hint = '请确认 EasyTier 守护进程已运行、RPC 地址正确，且 easytier-cli 已安装并在 PATH 中。'
    status.peers = []
    status.routes = []
    lastStatusUpdated.value = new Date().toLocaleTimeString()
  } finally {
    statusLoading.value = false
  }
}

async function loadDaemonStatus() {
  try {
    const { data } = await api.get(profilePath('/daemon/status'))
    daemonStatus.running = !!data?.running
    daemonStatus.pid = data?.pid || 0
    daemonStatus.last_start_error = data?.last_start_error || ''
    daemonStatus.started_version = data?.started_version || ''
    daemonModeEnabled.value = !!data?.daemon_mode_enabled
  } catch (_) {
    daemonStatus.running = false
    daemonStatus.pid = 0
    daemonStatus.last_start_error = ''
    daemonStatus.started_version = ''
    daemonModeEnabled.value = false
  }
}

async function loadDaemonLogs() {
  if (!daemonModeEnabled.value) return
  daemonLogsLoading.value = true
  try {
    const { data } = await api.get(profilePath('/daemon/logs'))
    daemonLogs.value = data?.logs ?? ''
  } catch (_) {
    daemonLogs.value = ''
  } finally {
    daemonLogsLoading.value = false
  }
}

async function startDaemon() {
  const imageTag = selectedVersion.value || form.image_tag || ''
  try {
    const { data } = await api.post(profilePath('/daemon/start'), { image_tag: imageTag || undefined })
    notifySuccess('已启动', data?.message || 'EasyTier daemon 已启动。')
    await loadDaemonStatus()
    await loadStatus()
    await loadDaemonLogs()
  } catch (e) {
    const err = e?.response?.data
    const msg = err?.error || e?.message || '启动失败'
    const hint = err?.hint || ''
    notifyError(msg, hint || undefined)
    await loadDaemonStatus()
  }
}

async function stopDaemon() {
  try {
    const { data } = await api.post(profilePath('/daemon/stop'))
    notifySuccess('已停止', data?.message || 'EasyTier daemon 已停止。')
    await loadDaemonStatus()
    await loadStatus()
    await loadDaemonLogs()
  } catch (_) {}
}

async function restartDaemon() {
  const imageTag = selectedVersion.value || form.image_tag || ''
  try {
    const { data } = await api.post(profilePath('/daemon/restart'), { image_tag: imageTag || undefined })
    notifySuccess('已重启', data?.message || 'EasyTier daemon 已重启。')
    await loadDaemonStatus()
    await loadStatus()
    await loadDaemonLogs()
  } catch (e) {
    const err = e?.response?.data
    const msg = err?.error || e?.message || '重启失败'
    const hint = err?.hint || ''
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
    const { data } = await api.post(profilePath('/daemon/release-port'))
    const msg = data?.message || '已解除端口占用。'
    if (data?.killed) {
      const portsStr = (data.ports_freed || []).length ? ` 端口 ${(data.ports_freed || []).join(', ')}` : ''
      notifySuccess('解除端口占用', `${msg} 已结束 ${data.killed} 个进程${portsStr}。`)
    } else {
      notifySuccess('解除端口占用', msg)
    }
    await loadDaemonStatus()
  } catch (e) {
    const err = e?.response?.data
    notifyError(err?.error || e?.message || '解除端口占用失败')
  } finally {
    releasePortLoading.value = false
  }
}

async function installRuntime() {
  const version = selectedVersion.value || form.image_tag?.trim()
  if (!version) {
    notifyError('请选择版本', '请先从下拉框选择要下载的 EasyTier 版本。')
    return
  }
  if (!selectedPlatformKey.value) {
    notifyError('请选择平台', '请先从下拉框选择目标平台。')
    return
  }
  const [osVal, arch] = selectedPlatformKey.value.split('/')
  if (!osVal || !arch) {
    notifyError('平台格式错误', '请重新选择平台。')
    return
  }
  runtimeInstalling.value = true
  runtimeError.value = ''
  try {
    const { data } = await api.post('/easytier/runtime/install', {
      version: version === 'latest' ? (releasesList.value[0]?.tag_name || 'latest') : version,
      os: osVal,
      arch,
    })
    if (data?.installed) {
      runtimeVersion.value = data.version || version
      notifySuccess('运行时已准备就绪', `已为 ${selectedPlatformKey.value} 安装 EasyTier ${data.version || version}。`)
      runtimeError.value = ''
      await loadRuntimeInstalled()
      await loadInstalledList()
    }
  } catch (e) {
    const message =
      (e?.response?.data && (e.response.data.error || e.response.data.warning)) ||
      e?.message ||
      '下载 EasyTier 运行时失败。'
    runtimeError.value = `${message} 如当前官方未提供该平台的守护进程，可考虑手动安装并通过 SKYLINK_EASYTIER_DAEMON_PATH 指定路径。`
  } finally {
    runtimeInstalling.value = false
  }
}

async function removeRuntime() {
  const version = selectedVersion.value || form.image_tag?.trim()
  if (!version || !selectedPlatformKey.value) return
  const [osVal, arch] = selectedPlatformKey.value.split('/')
  if (!osVal || !arch) return
  runtimeRemoving.value = true
  runtimeError.value = ''
  try {
    await api.delete('/easytier/runtime', {
      data: { version, os: osVal, arch },
    })
    notifySuccess('已移除', `已移除 ${selectedPlatformKey.value} 的 EasyTier ${version} 运行时。`)
    await loadRuntimeInstalled()
    await loadInstalledList()
  } catch (e) {
    const message = e?.response?.data?.error || e?.message || '移除失败。'
    runtimeError.value = message
  } finally {
    runtimeRemoving.value = false
  }
}

async function checkUpdate() {
  versionCheckLoading.value = true
  try {
    const { data } = await api.get('/easytier/version/check')
    versionCheck.current_version = data?.current_version || ''
    versionCheck.latest_version = data?.latest_version || ''
    versionCheck.update_available = !!data?.update_available
    versionCheck.release_url = data?.release_url || ''
  } catch (_) {}
  finally {
    versionCheckLoading.value = false
  }
}

function useLatestVersion() {
  if (versionCheck.latest_version) {
    form.image_tag = versionCheck.latest_version
    notifySuccess('已填入', '请保存配置并重启 EasyTier 守护进程以完成升级。')
  }
}

async function loadEasyTierAutostart() {
  try {
    const { data } = await api.get('/easytier/settings')
    easytierAutostart.value = !!data?.autostart_on_startup
  } catch (_) {
    easytierAutostart.value = false
  }
}

async function loadCliRaw() {
  cliRawLoading.value = true
  cliRawMetaError.value = ''
  cliRawStdout.value = ''
  cliRawStderr.value = ''
  try {
    const { data } = await api.get(profilePath('/cli-output'), {
      params: { target: cliRawTarget.value },
      silentError: true,
    } as any)
    if (data?.hint && data?.ok === false) {
      cliRawMetaError.value = data.hint
      return
    }
    cliRawStdout.value = typeof data?.stdout === 'string' ? data.stdout : ''
    cliRawStderr.value = typeof data?.stderr === 'string' ? data.stderr : ''
    if (data?.error) {
      cliRawMetaError.value = String(data.error)
    }
  } catch (e: any) {
    cliRawMetaError.value = e?.response?.data?.error || e?.message || String(e)
  } finally {
    cliRawLoading.value = false
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
  if (!selectedVersion.value) {
    if (form.image_tag && releasesList.value.some((r) => r.tag_name === form.image_tag)) {
      selectedVersion.value = form.image_tag
    } else if (releasesList.value.length) {
      selectedVersion.value = releasesList.value[0].tag_name
      form.image_tag = releasesList.value[0].tag_name
    }
  }
  if (!selectedPlatformKey.value && platformOptions.value.length) {
    const cur = platform.os && platform.arch ? `${platform.os}/${platform.arch}` : ''
    const match = cur ? platformOptions.value.find((o) => o.value === cur) : undefined
    selectedPlatformKey.value = match?.value ?? platformOptions.value[0].value
  }
  await loadRuntimeInstalled()
  await loadInstalledList()
})
  return {
    formRef,
    form,
    daemonStatus,
    daemonModeEnabled,
    networkMode,
    publicServer,
    showAdvanced,
    saving,
    statusLoading,
    versionCheckLoading,
    lastStatusUpdated,
    status,
    versionCheck,
    platform,
    releasesList,
    releasesError,
    platformsList,
    currentPlatformLabel,
    easytierHostSupported,
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
    daemonLogs,
    daemonLogsLoading,
    releasePortLoading,
    easytierAutostart,
    easytierAutostartSaving,
    onToggleEasyTierAutostart,
    cliRawTarget,
    cliRawStdout,
    cliRawStderr,
    cliRawMetaError,
    cliRawLoading,
    loadCliRaw,
    profiles,
    activeProfileId,
    newProfileName,
    profileOptions,
    canDeleteProfile,
    aggregatedErrors,
    installedOptionsForStatus,
    selectedInstalledKey,
    displayNodes,
    nodeTableColumns,
    routeColumns,
    installedListColumns,
    onDHCPChange,
    onIPv4Change,
    loadConfig,
    loadProfiles,
    onProfileChange,
    createProfile,
    deleteCurrentProfile,
    loadPlatform,
    loadReleases,
    loadPlatforms,
    loadRuntimeInstalled,
    loadInstalledList,
    onVersionOrPlatformChange,
    save,
    saveAndRestart,
    loadStatus,
    loadDaemonStatus,
    loadDaemonLogs,
    stopDaemon,
    startOrRestartDaemon,
    releasePort,
    installRuntime,
    removeRuntime,
    checkUpdate,
    useLatestVersion,
  }
}
