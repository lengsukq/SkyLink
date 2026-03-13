<template>
  <div>
    <page-header
      title="EasyTier"
      description="配置并查看 mesh VPN 状态；映射时可选择 mesh 内节点作为后端。推荐由 SkyLink 直接拉起并管理 EasyTier 守护进程，实现一站式使用。"
    />

    <n-alert type="info" class="page-section" style="margin-bottom: 0">
      <template #header>推荐步骤</template>
      ① 在「版本与运行时」选择版本与平台并下载 → ② 在「网络配置」填写网络名、密钥等并启用、保存 → ③ 在本卡片选择已下载版本并点击「启动」。
    </n-alert>

    <n-card title="状态" class="page-section page-card">
      <n-space vertical :size="12">
        <n-alert
          v-if="aggregatedErrors.length"
          type="error"
          :title="aggregatedErrors.length === 1 ? aggregatedErrors[0] : '本页错误与警告'"
        >
          <template v-if="aggregatedErrors.length > 1">
            <ul style="margin: 0; padding-left: 1.2em;">
              <li v-for="(msg, i) in aggregatedErrors" :key="i">{{ msg }}</li>
            </ul>
          </template>
        </n-alert>
        <n-space align="center" flex-wrap="wrap">
          <n-button size="small" :loading="statusLoading" @click="loadStatus">刷新</n-button>
          <span class="status-version">已启动版本: {{ daemonStatus.started_version || status.version || '—' }}</span>
          <span v-if="status.self_ipv4" class="status-self">本机 mesh IP: {{ status.self_ipv4 }}</span>
          <span v-if="lastStatusUpdated" class="status-updated">上次刷新时间: {{ lastStatusUpdated }}</span>
          <span
            v-if="status.peers && status.routes"
            class="status-hint status-summary"
          >
            Peers: {{ status.peers.length || 0 }} / Routes: {{ status.routes.length || 0 }}
          </span>
        </n-space>
        <n-space align="center" style="margin-top: 8px" flex-wrap="wrap">
          <span class="status-hint">已下载版本：</span>
          <n-select
            v-model:value="selectedInstalledKey"
            :options="installedOptionsForStatus"
            placeholder="选择要启动的版本"
            clearable
            style="min-width: 180px"
            size="small"
          />
          <span class="status-hint">
            Daemon 状态：
            <strong v-if="daemonModeEnabled && daemonStatus.running">运行中 (PID {{ daemonStatus.pid || '未知' }})</strong>
            <span v-else-if="daemonModeEnabled">未运行</span>
            <span v-else>—</span>
          </span>
          <n-button
            size="tiny"
            :disabled="!daemonModeEnabled"
            :title="daemonModeEnabled ? '' : '需在配置中开启 easytier.daemon_enabled 并重启 SkyLink'"
            @click="startDaemon"
          >启动</n-button>
          <n-button
            size="tiny"
            :disabled="!daemonModeEnabled || !daemonStatus.running"
            :title="!daemonModeEnabled ? '需在配置中开启 easytier.daemon_enabled 并重启 SkyLink' : (!daemonStatus.running ? '守护进程未运行' : '')"
            @click="stopDaemon"
          >停止</n-button>
          <n-button
            size="tiny"
            :disabled="!daemonModeEnabled"
            :title="daemonModeEnabled ? '' : '需在配置中开启 easytier.daemon_enabled 并重启 SkyLink'"
            @click="restartDaemon"
          >重启</n-button>
          <n-button
            size="tiny"
            :loading="releasePortLoading"
            title="结束占用 EasyTier 相关端口的进程（RPC 15888、listeners 11010–11013 等），便于重新启动"
            @click="releasePort"
          >解除端口占用</n-button>
          <span v-if="!daemonModeEnabled" class="status-hint">
            请在配置中开启 <code>easytier.daemon_enabled</code> 并重启 SkyLink 后，即可在此处控制守护进程。
          </span>
        </n-space>
        <n-divider v-if="daemonModeEnabled" style="margin: 12px 0 8px 0">守护进程日志</n-divider>
        <n-space v-if="daemonModeEnabled" vertical size="small">
          <n-button size="tiny" :loading="daemonLogsLoading" @click="loadDaemonLogs">刷新</n-button>
          <pre class="daemon-logs">{{ daemonLogs || '（暂无日志，启动守护进程后点击刷新）' }}</pre>
        </n-space>
        <div v-if="!form.enabled" class="status-hint">
          <span v-if="daemonModeEnabled">
            EasyTier 当前未启用。请在下方开启“启用”并保存配置，然后使用上方按钮启动 EasyTier 守护进程。
          </span>
          <span v-else>
            EasyTier 当前未启用。请在下方开启“启用”并保存配置，并确保已在本机启动 EasyTier 守护进程或在配置中开启守护进程模式。
          </span>
        </div>
        <div v-else-if="status.ok && !status.peers.length && !status.routes.length" class="status-hint">
          EasyTier 已运行，但当前没有任何 peers 或路由；这通常表示你是第一个节点，或尚未成功加入到目标网络。
        </div>
        <template v-if="displayNodes.length">
          <div class="status-hint" style="margin-bottom: 8px">
            节点信息 {{ displayNodes.length }}
          </div>
          <div class="status-table-wrapper">
            <n-data-table
              :columns="nodeTableColumns"
              :data="displayNodes"
              :bordered="false"
              size="small"
            />
          </div>
        </template>
        <n-collapse v-if="status.routes && status.routes.length">
          <n-collapse-item title="路由表" name="routes">
            <div class="status-table-wrapper">
              <n-data-table :columns="routeColumns" :data="status.routes" :bordered="false" size="small" />
            </div>
          </n-collapse-item>
        </n-collapse>
      </n-space>
    </n-card>

    <n-grid cols="1 s:1 m:2 l:2 xl:3" x-gap="16" y-gap="16" class="page-section">
      <n-gi>
        <n-card title="版本与运行时" class="page-card">
          <n-space vertical size="small">
            <n-form-item label="EasyTier 版本">
              <n-select
                v-model:value="selectedVersion"
                :options="releaseOptions"
                placeholder="从 GitHub Releases 选择"
                filterable
                clearable
                style="width: 100%"
                @update:value="onVersionOrPlatformChange"
              />
            </n-form-item>
            <n-form-item label="平台">
              <n-select
                v-model:value="selectedPlatformKey"
                :options="platformOptions"
                placeholder="选择平台"
                style="width: 100%"
                @update:value="onVersionOrPlatformChange"
              />
            </n-form-item>
            <n-space align="center">
              <n-button :loading="runtimeInstalling" @click="installRuntime">
                下载 EasyTier 运行时
              </n-button>
              <n-button :loading="runtimeRemoving" :disabled="!runtimeInstalled" @click="removeRuntime">
                移除
              </n-button>
            </n-space>
            <span v-if="runtimeInstalled" class="runtime-hint">已安装：当前所选版本与平台。启动/停止/重启请在顶部状态卡片选择已下载版本后操作。<span v-if="!daemonModeEnabled">（需先开启守护进程模式并重启 SkyLink）</span></span>
            <span v-else-if="selectedVersion && selectedPlatformKey" class="runtime-hint">未安装</span>
            <span v-if="releasesError" class="status-error">{{ releasesError }}</span>
            <span v-if="runtimeError" class="status-error">{{ runtimeError }}</span>
            <span v-if="platformError" class="status-error">{{ platformError }}</span>
            <n-space align="center" style="margin-top: 8px">
              <n-button :loading="versionCheckLoading" size="tiny" @click="checkUpdate">检查更新</n-button>
              <span v-if="versionCheck.latest_version" class="easytier-version-hint">
                最新: {{ versionCheck.latest_version }}
                <a v-if="versionCheck.release_url" :href="versionCheck.release_url" target="_blank" rel="noopener">Release</a>
                <n-button v-if="versionCheck.update_available" text type="primary" size="tiny" @click="useLatestVersion">使用此版本</n-button>
              </span>
            </n-space>
            <n-divider style="margin: 12px 0 8px 0">已下载的版本</n-divider>
            <div v-if="installedList.length" class="status-table-wrapper">
              <n-data-table
                :columns="installedListColumns"
                :data="installedList"
                :bordered="false"
                size="small"
              />
            </div>
            <span v-else class="runtime-hint">暂无已下载的运行时，请在上方选择版本与平台后点击「下载 EasyTier 运行时」。</span>
          </n-space>
        </n-card>
      </n-gi>

      <n-gi>
        <n-card title="网络配置" class="page-card">
          <div class="status-hint" style="margin-bottom: 8px">
            启用后，可通过顶部状态卡片控制 EasyTier 守护进程运行，并在本页查看 mesh 状态与 WireGuard 配置。
          </div>
          <n-form ref="formRef" :model="form" label-placement="left" label-width="120">
            <n-form-item label="网络名">
              <n-input v-model:value="form.network_name" placeholder="网络标识（ET_NETWORK_NAME，对应 EasyTier network_identity.name）" />
            </n-form-item>
            <n-form-item label="网络密钥">
              <n-input
                v-model:value="form.network_secret"
                type="password"
                show-password-on="click"
                placeholder="网络密钥（ET_NETWORK_SECRET，对应 EasyTier network_identity.secret）"
              />
            </n-form-item>
            <n-form-item label="虚拟 IPv4 地址">
              <n-space align="center">
                <n-input
                  v-model:value="form.ipv4"
                  class="ipv4-input"
                  placeholder="如 10.0.111.99（mesh 内 IP）"
                  :disabled="form.dhcp"
                  @update:value="onIPv4Change"
                />
                <span class="cidr-suffix">/24</span>
                <n-checkbox
                  v-model:checked="form.dhcp"
                  @update:checked="onDHCPChange"
                >
                  DHCP
                </n-checkbox>
              </n-space>
            </n-form-item>
            <n-form-item label="网络方式">
              <n-radio-group v-model:value="networkMode">
                <n-space>
                  <n-radio value="public">公共服务器</n-radio>
                  <n-radio value="manual">手动</n-radio>
                  <n-radio value="standalone">独立</n-radio>
                </n-space>
              </n-radio-group>
            </n-form-item>
            <n-form-item v-if="networkMode === 'public'" label="公共服务器">
              <n-input
                v-model:value="publicServer"
                placeholder="如 tcp://public.easytier.cn:11010/IPv4"
                @update:value="(v) => (form.peers = v)"
              />
              <template #feedback>
                <span class="status-hint">
                  示例：tcp://8.8.8.8:11010，输入后在下拉框中选择生效。
                </span>
              </template>
            </n-form-item>
            <n-form-item
              v-else-if="networkMode === 'manual'"
              label="初始节点（peers）"
            >
              <n-input
                v-model:value="form.peers"
                type="textarea"
                :rows="2"
                placeholder="如 tcp://public.easytier.cn:11010，多个用逗号或换行（对应 ET_PEERS / peers[].uri）"
              />
            </n-form-item>
            <n-form-item v-else label="初始节点（peers）">
              <n-input
                type="textarea"
                :rows="2"
                disabled
                placeholder="独立模式下无需填写初始节点，将作为第一个节点加入网络。"
              />
            </n-form-item>
            <n-divider />
            <n-form-item label="主机名">
              <n-input v-model:value="form.hostname" placeholder="ET_HOSTNAME，留空则按默认行为" />
            </n-form-item>
            <n-form-item label="公网发现节点（external-node）">
              <n-input
                v-model:value="form.external_node"
                placeholder="如 tcp://public.easytier.cn:11010（ET_EXTERNAL_NODE，仅用于公网发现，可选）"
              />
            </n-form-item>
            <n-form-item label="子网代理（proxy-networks）">
              <n-input
                v-model:value="form.proxy_networks"
                type="textarea"
                :rows="2"
                placeholder="如 10.0.0.0/24，多个子网用逗号或换行（ET_PROXY_NETWORKS / vpn_portal_config.proxy_networks）"
              />
            </n-form-item>
            <n-form-item label="启用">
              <n-switch v-model:value="form.enabled" />
            </n-form-item>
            <n-form-item v-if="showAdvanced" label="RPC 地址">
              <n-input v-model:value="form.rpc_portal" placeholder="SkyLink 连接 EasyTier 的地址，如 easytier:15888" />
            </n-form-item>
            <n-form-item v-if="showAdvanced" label="VPN Portal（WireGuard）">
              <n-input
                v-model:value="form.vpn_portal"
                placeholder="如 wg://0.0.0.0:11013/10.14.14.0/24"
              />
              <template #feedback>
                <span class="status-hint">启用并保存后，可在本页下方「WireGuard 客户端配置」卡片中获取客户端配置。</span>
              </template>
            </n-form-item>
          </n-form>
          <n-space>
            <n-button type="primary" :loading="saving" @click="save">保存配置</n-button>
            <n-button :loading="saving" @click="saveAndRestart">保存并重启</n-button>
            <n-button @click="showAdvanced = !showAdvanced">{{ showAdvanced ? '收起高级' : '高级' }}</n-button>
          </n-space>
        </n-card>
      </n-gi>
    </n-grid>

    <n-card title="WireGuard 客户端配置（VPN Portal）" class="page-section page-card">
      <n-space vertical :size="8">
        <div class="status-hint">
          需先在上方网络配置的「高级」中启用并保存 VPN Portal，再点击获取配置。
        </div>
        <n-space align="center" :size="8">
          <n-button size="small" :loading="vpnPortalLoading" @click="loadVPNPortalConfig">获取配置</n-button>
          <n-button size="small" :disabled="!vpnPortal.config" @click="copyVPNPortalConfig">复制</n-button>
        </n-space>
        <div v-if="vpnPortal.error" class="status-error">
          {{ vpnPortal.error }}
        </div>
        <n-input
          v-model:value="vpnPortal.config"
          type="textarea"
          :rows="8"
          readonly
          placeholder="在 EasyTier 页配置并启用 VPN Portal，重启 EasyTier 后点击“获取配置”以显示 WireGuard 客户端配置。"
        />
      </n-space>
    </n-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, h, computed } from 'vue'
import { NCard, NForm, NFormItem, NInput, NInputGroup, NButton, NSpace, NDataTable, NCollapse, NCollapseItem, NSwitch, NRadioGroup, NRadio, NCheckbox, NDivider, NGrid, NGi, NSelect, NAlert } from 'naive-ui'
import api from '../api/client'
import { notifySuccess, notifyError } from '../ui/notify'
import PageHeader from '../components/PageHeader.vue'

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
  vpn_portal: '',
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
const vpnPortalLoading = ref(false)
const lastStatusUpdated = ref('')

const status = reactive({
  ok: false,
  error: '',
  hint: '',
  version: '',
  self_ipv4: '',
  self_hostname: '',
  peers: [],
  routes: [],
})
const vpnPortal = reactive({
  config: '',
  error: '',
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
const releasesList = ref([])
const releasesError = ref('')
const platformsList = ref([])
const currentPlatformLabel = ref('')
const releaseOptions = ref([])
const platformOptions = ref([])
const selectedVersion = ref(null)
const selectedPlatformKey = ref(null)
const runtimeInstalling = ref(false)
const runtimeRemoving = ref(false)
const runtimeInstalled = ref(false)
const runtimeVersion = ref('')
const platformError = ref('')
const runtimeError = ref('')
const installedList = ref([])
const removingInstalledKey = ref('')
const daemonLogs = ref('')
const daemonLogsLoading = ref(false)
const releasePortLoading = ref(false)

const aggregatedErrors = computed(() => {
  const list = []
  if (daemonStatus.last_start_error) list.push(`上次启动错误：${daemonStatus.last_start_error}`)
  if (status.error) list.push(status.error)
  if (releasesError.value) list.push(releasesError.value)
  if (runtimeError.value) list.push(runtimeError.value)
  if (platformError.value) list.push(platformError.value)
  if (vpnPortal.error) list.push(vpnPortal.error)
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
  const list = []
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
    render: (row) => (row.latency_ms != null && row.latency_ms !== '' ? `${Number(row.latency_ms)}ms` : '—'),
  },
  { title: '内核版本', key: 'version', width: 120, ellipsis: true },
]
const routeColumns = [
  { title: 'IPv4', key: 'ipv4', width: 120 },
  { title: 'Hostname', key: 'hostname', ellipsis: true },
  { title: 'Proxy CIDRs', key: 'proxy_cidrs', ellipsis: true },
  { title: 'NextHop', key: 'next_hop_ipv4', width: 120 },
]

function installedItemKey(row) {
  return `${row.version}-${row.os}-${row.arch}`
}

function isCurrentSelected(row) {
  const plat = selectedPlatformKey.value ? `${row.os}/${row.arch}` === selectedPlatformKey.value : false
  const ver = (selectedVersion.value || form.image_tag) && row.version === (selectedVersion.value || form.image_tag)
  return plat && ver
}

const installedListColumns = [
  { title: '版本', key: 'version', width: 100 },
  { title: '平台', key: 'platform', width: 110, render: (row) => `${row.os}/${row.arch}` },
  { title: '说明', key: 'current', width: 80, render: (row) => (isCurrentSelected(row) ? '当前使用' : '') },
  {
    title: '操作',
    key: 'action',
    width: 80,
    render: (row) => {
      const key = installedItemKey(row)
      return h(
        NButton,
        {
          size: 'tiny',
          tertiary: true,
          loading: removingInstalledKey.value === key,
          disabled: removingInstalledKey.value !== '' && removingInstalledKey.value !== key,
          onClick: () => removeInstalledItem(row),
        },
        { default: () => '移除' }
      )
    },
  },
]

function onDHCPChange(val) {
  form.dhcp = val
  if (val) {
    form.ipv4 = ''
  }
}

function onIPv4Change(val) {
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
      form.vpn_portal = data.vpn_portal || ''
      const peers = (data.peers || '').trim()
      form.peers = peers
      if (!peers) {
        networkMode.value = 'standalone'
      } else if (peers.includes('public.easytier.cn')) {
        networkMode.value = 'public'
        const first = peers
          .split(/[\n,]+/)
          .map((s) => s.trim())
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

async function loadPlatform() {
  try {
    const { data } = await api.get('/easytier/platform')
    platform.os = data?.os || ''
    platform.arch = data?.arch || ''
    platform.label = data?.label || ''
    platformError.value = ''
  } catch (e) {
    platform.os = ''
    platform.arch = ''
    platform.label = ''
    platformError.value = '无法获取后端平台信息，运行时下载功能可能不可用。'
  }
}

async function loadReleases() {
  releasesError.value = ''
  try {
    const { data } = await api.get('/easytier/releases')
    const list = data?.releases || []
    releasesList.value = list
    releaseOptions.value = list.map((r) => ({ label: r.tag_name, value: r.tag_name }))
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
    platformOptions.value = list.map((p) => ({ label: p.label, value: p.label }))
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

async function removeInstalledItem(row) {
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
      const { data } = await api.post('/easytier/daemon/restart')
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
    const { data } = await api.get('/easytier/status')
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
    const { data } = await api.get('/easytier/daemon/status')
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
    const { data } = await api.get('/easytier/daemon/logs')
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
    const { data } = await api.post('/easytier/daemon/start', { image_tag: imageTag || undefined })
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
    const { data } = await api.post('/easytier/daemon/stop')
    notifySuccess('已停止', data?.message || 'EasyTier daemon 已停止。')
    await loadDaemonStatus()
    await loadStatus()
    await loadDaemonLogs()
  } catch (_) {}
}

async function restartDaemon() {
  const imageTag = selectedVersion.value || form.image_tag || ''
  try {
    const { data } = await api.post('/easytier/daemon/restart', { image_tag: imageTag || undefined })
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

async function releasePort() {
  releasePortLoading.value = true
  try {
    const { data } = await api.post('/easytier/daemon/release-port')
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

async function loadVPNPortalConfig() {
  vpnPortalLoading.value = true
  try {
    const { data } = await api.get('/easytier/vpn-portal')
    vpnPortal.config = data?.config || ''
    vpnPortal.error = data?.error || ''
    if (!vpnPortal.config && !vpnPortal.error) {
      vpnPortal.error = '未获取到 WireGuard 配置，可能未在高级配置中启用 VPN Portal，或 EasyTier 未启动/未加入网络。'
    }
  } catch (_) {
    vpnPortal.config = ''
    vpnPortal.error = '获取 WireGuard 配置失败，可能未启用 VPN Portal 或 EasyTier 未启动 / RPC 地址不可达。'
  } finally {
    vpnPortalLoading.value = false
  }
}

async function copyVPNPortalConfig() {
  if (!vpnPortal.config) return
  try {
    await navigator.clipboard.writeText(vpnPortal.config)
    notifySuccess('已复制', 'WireGuard 配置已复制到剪贴板。')
  } catch (_) {
    notifyError('复制失败', '无法复制到剪贴板，请手动选择并复制。')
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

onMounted(async () => {
  loadPlatform()
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
  if (!selectedPlatformKey.value) {
    selectedPlatformKey.value = currentPlatformLabel.value || (platformOptions.value[0]?.value ?? null)
  }
  await loadRuntimeInstalled()
  await loadInstalledList()
})
</script>

<style scoped>
.easytier-version-hint {
  font-size: 12px;
  color: #666;
}
.easytier-version-hint a {
  margin-left: 8px;
}
.status-version,
.status-self {
  font-size: 13px;
  color: #666;
}
.status-self {
  margin-left: 12px;
}
.status-error {
  color: var(--n-error-color);
  font-size: 13px;
}
.status-hint {
  font-size: 13px;
  color: #666;
}
.status-hint code {
  font-family: var(--n-font-mono, monospace);
  font-size: 12px;
  padding: 0 4px;
  background: var(--n-color-target, #f0f0f0);
  border-radius: 3px;
}
.status-updated {
  margin-left: 12px;
  font-size: 12px;
  color: #999;
}
.runtime-hint {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: #666;
}
.field-label {
  width: 120px;
  font-size: 13px;
  color: #666;
}
.daemon-logs {
  margin: 0;
  padding: 10px;
  max-height: 280px;
  overflow: auto;
  font-size: 12px;
  line-height: 1.4;
  background: var(--n-color-target, #f5f5f5);
  border-radius: 6px;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
