<template>
  <div>
    <page-header
      title="EasyTier"
      description="配置并查看 mesh VPN 状态；映射时可选择 mesh 内节点作为后端。配置保存后会写入 data/easytier.env，请重启 EasyTier（容器或守护进程）使配置生效。"
    />

    <n-card title="状态" class="page-section page-card">
      <n-space vertical :size="12">
        <n-space align="center">
          <n-button size="small" :loading="statusLoading" @click="loadStatus">刷新</n-button>
          <span v-if="status.version" class="status-version">当前运行版本: {{ status.version }}</span>
          <span v-if="status.self_ipv4" class="status-self">本机 mesh IP: {{ status.self_ipv4 }}</span>
          <span v-if="lastStatusUpdated" class="status-updated">上次刷新时间: {{ lastStatusUpdated }}</span>
          <span
            v-if="status.peers && status.routes"
            class="status-hint status-summary"
          >
            Peers: {{ status.peers.length || 0 }} / Routes: {{ status.routes.length || 0 }}
          </span>
        </n-space>
        <n-space align="center" style="margin-top: 4px">
          <template v-if="daemonModeEnabled">
            <span class="status-hint">
              Daemon 状态：
              <strong v-if="daemonStatus.running">运行中 (PID {{ daemonStatus.pid || '未知' }})</strong>
              <span v-else>未运行</span>
            </span>
            <n-button size="tiny" @click="startDaemon">启动</n-button>
            <n-button size="tiny" @click="stopDaemon" :disabled="!daemonStatus.running">停止</n-button>
            <n-button size="tiny" @click="restartDaemon">重启</n-button>
          </template>
          <template v-else>
            <span class="status-hint">
              当前未启用 EasyTier 守护进程模式，通常表示你通过 Docker / docker compose 管理 EasyTier 容器。
              请在下方完成版本与网络配置后，通过容器编排系统启动或重启 EasyTier。
            </span>
          </template>
        </n-space>
        <div v-if="daemonStatus.last_start_error" class="status-error">
          上次启动错误：{{ daemonStatus.last_start_error }}
        </div>
        <div v-if="!form.enabled" class="status-hint">
          <span v-if="daemonModeEnabled">
            EasyTier 当前未启用。请在下方开启“启用”并保存配置，然后使用上方按钮启动 EasyTier 守护进程。
          </span>
          <span v-else>
            EasyTier 当前未启用。请在下方开启“启用”并保存配置，然后在 Docker / docker compose 中启动或重启 EasyTier 容器。
          </span>
        </div>
        <div v-else-if="status.error" class="status-error">
          EasyTier 已启用，但状态获取失败：{{ status.error }}。请检查 EasyTier 是否正在运行以及 RPC 地址配置是否正确。
        </div>
        <div v-else-if="status.ok && !status.peers.length && !status.routes.length" class="status-hint">
          EasyTier 已运行，但当前没有任何 peers 或路由；这通常表示你是第一个节点，或尚未成功加入到目标网络。
        </div>
        <div class="status-table-wrapper">
          <n-data-table
            v-if="status.peers && status.peers.length"
            :columns="peerColumns"
            :data="status.peers"
            :bordered="false"
            size="small"
          />
        </div>
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
            <n-space align="center">
              <span class="field-label">EasyTier 版本（镜像 tag）</span>
              <n-input-group style="flex: 1">
                <n-input v-model:value="form.image_tag" placeholder="如 v2.2.3 或 latest" />
                <n-button :loading="versionCheckLoading" @click="checkUpdate">检查更新</n-button>
              </n-input-group>
            </n-space>
            <span v-if="versionCheck.latest_version" class="easytier-version-hint">
              当前配置: {{ versionCheck.current_version || form.image_tag || '未设置' }}；最新: {{ versionCheck.latest_version }}
              <a v-if="versionCheck.release_url" :href="versionCheck.release_url" target="_blank" rel="noopener">Release</a>
              <n-button v-if="versionCheck.update_available" text type="primary" size="tiny" @click="useLatestVersion">使用此版本</n-button>
            </span>
            <span class="runtime-hint">
              当前后端平台: {{ platform.label || '未知' }}
              <n-button text size="tiny" :loading="runtimeInstalling" @click="installRuntime">
                下载/更新 EasyTier 运行时
              </n-button>
              <span v-if="runtimeVersion">（已安装: {{ runtimeVersion }}）</span>
            </span>
            <span v-if="runtimeError" class="status-error">
              {{ runtimeError }}
            </span>
            <span v-if="platformError" class="status-error">
              {{ platformError }}
            </span>
          </n-space>
        </n-card>
      </n-gi>

      <n-gi>
        <n-card title="网络配置" class="page-card">
          <div class="status-hint" style="margin-bottom: 8px">
            启用后，可通过顶部状态卡片控制 EasyTier 运行，并在本页查看 mesh 状态与 WireGuard 配置。
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
                />
                <span class="cidr-suffix">/24</span>
                <n-checkbox v-model:checked="form.dhcp">DHCP</n-checkbox>
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
            </n-form-item>
          </n-form>
          <n-space>
            <n-button type="primary" :loading="saving" @click="save">保存配置</n-button>
            <n-button @click="showAdvanced = !showAdvanced">{{ showAdvanced ? '收起高级' : '高级' }}</n-button>
          </n-space>
        </n-card>
      </n-gi>
    </n-grid>

    <n-card title="WireGuard 客户端配置（VPN Portal）" class="page-section page-card">
      <n-space vertical :size="8">
        <div class="status-hint">
          需要在上方网络配置中启用 VPN Portal 后再获取配置。
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
import { ref, reactive, onMounted } from 'vue'
import { NCard, NForm, NFormItem, NInput, NInputGroup, NButton, NSpace, NDataTable, NCollapse, NCollapseItem, NSwitch, NRadioGroup, NRadio, NCheckbox, NDivider, NGrid, NGi } from 'naive-ui'
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
const runtimeInstalling = ref(false)
const runtimeVersion = ref('')
const platformError = ref('')
const runtimeError = ref('')

const peerColumns = [
  { title: 'IPv4', key: 'ipv4', width: 120 },
  { title: 'Hostname', key: 'hostname', ellipsis: true },
  { title: 'Version', key: 'version', width: 90 },
]
const routeColumns = [
  { title: 'IPv4', key: 'ipv4', width: 120 },
  { title: 'Hostname', key: 'hostname', ellipsis: true },
  { title: 'Proxy CIDRs', key: 'proxy_cidrs', ellipsis: true },
  { title: 'NextHop', key: 'next_hop_ipv4', width: 120 },
]

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
    notifySuccess('已保存', data?.message || '请重启 EasyTier 容器使配置生效。')
    await loadConfig()
  } finally {
    saving.value = false
  }
}

async function loadStatus() {
  statusLoading.value = true
  try {
    const { data } = await api.get('/easytier/status')
    status.ok = data?.ok ?? false
    status.error = data?.error || ''
    status.version = data?.version || ''
    status.self_ipv4 = data?.self_ipv4 || ''
    status.self_hostname = data?.self_hostname || ''
    status.peers = data?.peers || []
    status.routes = data?.routes || []
    lastStatusUpdated.value = new Date().toLocaleTimeString()
  } catch (_) {
    status.error = '无法获取 EasyTier 状态，可能未启用或 RPC 地址不可达。'
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
    daemonModeEnabled.value = !!data?.daemon_mode_enabled
  } catch (_) {
    daemonStatus.running = false
    daemonStatus.pid = 0
    daemonStatus.last_start_error = ''
    daemonModeEnabled.value = false
  }
}

async function startDaemon() {
  try {
    const { data } = await api.post('/easytier/daemon/start')
    notifySuccess('已启动', data?.message || 'EasyTier daemon 已启动。')
    await loadDaemonStatus()
    await loadStatus()
  } catch (_) {}
}

async function stopDaemon() {
  try {
    const { data } = await api.post('/easytier/daemon/stop')
    notifySuccess('已停止', data?.message || 'EasyTier daemon 已停止。')
    await loadDaemonStatus()
    await loadStatus()
  } catch (_) {}
}

async function restartDaemon() {
  await stopDaemon()
  await startDaemon()
}

async function installRuntime() {
  runtimeInstalling.value = true
  runtimeError.value = ''
  try {
    const body = {}
    if (form.image_tag && form.image_tag.trim()) {
      body.version = form.image_tag.trim()
    }
    const { data } = await api.post('/easytier/runtime/install', body)
    if (data?.installed) {
      runtimeVersion.value = data.version || ''
      notifySuccess('运行时已准备就绪', `已为 ${platform.label || '当前平台'} 安装 EasyTier 运行时。`)
      runtimeError.value = ''
    }
  } catch (e) {
    const message =
      (e?.response?.data && (e.response.data.error || e.response.data.warning)) ||
      e?.message ||
      '下载/更新 EasyTier 运行时失败。'
    runtimeError.value = `${message} 如当前官方未提供该平台的守护进程，可考虑手动安装并通过 SKYLINK_EASYTIER_DAEMON_PATH 指定路径。`
  } finally {
    runtimeInstalling.value = false
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
    notifySuccess('已填入', '请保存配置并重启 EasyTier 容器以完成升级。')
  }
}

onMounted(() => {
  loadPlatform()
  loadConfig()
  loadStatus()
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
</style>
