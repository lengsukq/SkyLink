<template>
  <div class="easytier-panel">
    <p class="easytier-tab-lead">
      Mesh VPN 与运行时管理；映射时可选用 mesh 内节点作为后端。完整集成仅在 <strong>Windows</strong> 版 SkyLink 上可用。
    </p>

    <n-alert v-if="!easytierHostSupported" type="warning" title="当前环境不支持 EasyTier 集成" class="panel-section">
      SkyLink 内置的 EasyTier（守护进程、下载 Windows 运行时、端口释放与 easytier-cli 状态）仅在 <strong>Windows 主机</strong>上可用。
      若你使用 Linux/Docker 等部署，本标签无法保存配置或拉起进程；请在 Windows 上运行 SkyLink，或自行在其他环境部署 EasyTier。
    </n-alert>

    <n-card v-if="easytierHostSupported" size="small" class="panel-section page-card">
      <n-space align="center" justify="space-between">
        <span>SkyLink 启动时自动启动 EasyTier</span>
        <n-switch
          :value="easytierAutostart"
          :loading="easytierAutostartSaving"
          @update:value="onToggleEasyTierAutostart"
        />
      </n-space>
    </n-card>

    <easy-tier-intro-alert />

    <easy-tier-profiles-card
      v-model:active-profile-id="activeProfileId"
      v-model:new-profile-name="newProfileName"
      :profile-options="profileOptions"
      :can-delete="canDeleteProfile"
      @profile-change="onProfileChange"
      @create="createProfile"
      @delete="deleteCurrentProfile"
      @open-runtime="openRuntimeModal"
    />

    <n-card title="状态" class="panel-section page-card">
      <n-space vertical :size="12">
        <n-alert
          v-if="aggregatedErrors.length"
          type="error"
          :title="aggregatedErrors.length === 1 ? aggregatedErrors[0] : '本页错误与警告'"
        >
          <template v-if="aggregatedErrors.length > 1">
            <ul class="easytier-error-list">
              <li v-for="(msg, i) in aggregatedErrors" :key="i">{{ msg }}</li>
            </ul>
          </template>
        </n-alert>
        <n-alert v-if="!hasNodeConfig" type="warning" title="尚未完成节点配置">
          当前实例还没有可用节点配置，运行操作已禁用。请先点击
          <n-button text type="primary" style="padding: 0 2px" @click="openNodeConfigModal">
            配置节点
          </n-button>
          ，完成网络名、密钥和初始节点后再启动/重启。
        </n-alert>
        <n-space align="center" flex-wrap="wrap">
          <n-button size="small" :loading="statusLoading" :disabled="runtimeOpsDisabled" :title="runtimeOpsDisabledReason" @click="loadStatus">刷新</n-button>
          <n-button
            size="small"
            type="primary"
            ghost
            @click="openNodeConfigModal"
          >配置节点</n-button>
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
            :disabled="runtimeOpsDisabled || !daemonStatus.running"
            :title="runtimeOpsDisabled ? runtimeOpsDisabledReason : (!daemonStatus.running ? '守护进程未运行' : '')"
            @click="stopDaemon"
          >停止</n-button>
          <n-button
            size="tiny"
            :disabled="runtimeOpsDisabled"
            :title="runtimeOpsDisabled ? runtimeOpsDisabledReason : (daemonStatus.running ? '重启 EasyTier 守护进程' : '启动 EasyTier 守护进程')"
            @click="startOrRestartDaemon"
          >{{ daemonStatus.running ? '重启' : '启动' }}</n-button>
          <n-button
            size="tiny"
            :disabled="runtimeOpsDisabled"
            :loading="releasePortLoading"
            :title="runtimeOpsDisabled ? runtimeOpsDisabledReason : '结束占用 EasyTier 相关端口的进程（RPC 15888、listeners 11010–11013 等），便于重新启动'"
            @click="releasePort"
          >解除端口占用</n-button>
          <span v-if="!daemonModeEnabled" class="status-hint">
            请在配置中开启 <code>easytier.daemon_enabled</code> 并重启 SkyLink 后，即可在此处控制守护进程。
          </span>
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
        <n-divider style="margin: 12px 0 8px 0">Peer 常驻信息</n-divider>
        <div class="status-hint" style="margin-bottom: 8px">
          页面停留期间每秒自动刷新一次（仅当前标签页可见时）。
        </div>
        <div v-if="peerResidentRows.length" class="status-table-wrapper">
          <n-data-table
            :columns="peerResidentColumns"
            :data="peerResidentRows"
            :bordered="false"
            size="small"
          />
        </div>
        <div v-else class="status-hint">
          暂无 peer 数据，待成功入网后会在这里持续显示。
        </div>
        <n-collapse v-if="status.routes && status.routes.length">
          <n-collapse-item title="路由表" name="routes">
            <div class="status-table-wrapper">
              <n-data-table :columns="routeColumns" :data="status.routes" :bordered="false" size="small" />
            </div>
          </n-collapse-item>
        </n-collapse>
        <n-divider style="margin: 12px 0 8px 0">在网信息（easytier-cli 原始输出）</n-divider>
        <p class="status-hint" style="margin-bottom: 8px">
          对应文档中的管理命令输出；用于核对 peers、路由与本机节点。
          <a href="https://easytier.cn/guide/network/configurations.html" target="_blank" rel="noopener noreferrer">完整配置选项</a>
          中的环境变量（如 <code>ET_RPC_PORTAL</code>）需与当前配置一致。
        </p>
        <n-space align="center" wrap>
          <n-select
            v-model:value="cliRawTarget"
            :options="cliRawTargetOptions"
            style="min-width: 180px"
            size="small"
            :disabled="!easytierHostSupported"
          />
          <n-button size="small" :loading="cliRawLoading" :disabled="!easytierHostSupported" @click="loadCliRaw">获取</n-button>
        </n-space>
        <div v-if="cliRawMetaError" class="status-error" style="margin-top: 10px">{{ cliRawMetaError }}</div>
        <div v-if="parsedCliPeerRows.length" class="peer-modern-view">
          <div class="peer-modern-stats">
            <div class="peer-stat-card">
              <div class="peer-stat-label">总节点数</div>
              <div class="peer-stat-value">{{ cliPeerSummary.total }}</div>
            </div>
            <div class="peer-stat-card">
              <div class="peer-stat-label">P2P</div>
              <div class="peer-stat-value peer-stat-success">{{ cliPeerSummary.p2p }}</div>
            </div>
            <div class="peer-stat-card">
              <div class="peer-stat-label">Relay</div>
              <div class="peer-stat-value peer-stat-warn">{{ cliPeerSummary.relay }}</div>
            </div>
            <div class="peer-stat-card">
              <div class="peer-stat-label">平均延迟</div>
              <div class="peer-stat-value">{{ cliPeerSummary.avgLatencyMs == null ? '—' : `${cliPeerSummary.avgLatencyMs} ms` }}</div>
            </div>
          </div>
          <div class="status-table-wrapper">
            <n-data-table
              :columns="cliPeerColumns"
              :data="parsedCliPeerRows"
              :bordered="false"
              size="small"
            />
          </div>
        </div>
        <template v-if="cliRawStdout || cliRawStderr">
          <div v-if="cliRawStdout" class="cli-raw-label">stdout</div>
          <pre v-if="cliRawStdout" class="cli-raw-block">{{ cliRawStdout }}</pre>
          <div v-if="cliRawStderr" class="cli-raw-label">stderr</div>
          <pre v-if="cliRawStderr" class="cli-raw-block cli-raw-err">{{ cliRawStderr }}</pre>
        </template>
        <p v-else-if="easytierHostSupported && !cliRawLoading && !cliRawMetaError" class="status-hint" style="margin-top: 8px">
          选择子命令后点击「获取」。需守护进程已运行；CLI 与 core 同目录（推荐）或可在 PATH 中找到 <code>easytier-cli</code>。
        </p>
        <n-divider v-if="daemonModeEnabled" style="margin: 12px 0 8px 0">守护进程日志</n-divider>
        <p v-if="daemonModeEnabled && easytierHostSupported" class="status-hint" style="margin: 0 0 8px 0">
          默认后台运行且不弹出控制台；日志会显示在此处。若需弹出独立控制台调试，请设置环境变量
          <code>SKYLINK_EASYTIER_DAEMON_CAPTURE_OUTPUT=0</code> 并重启 SkyLink。
        </p>
        <n-space v-if="daemonModeEnabled" vertical size="small">
          <n-button size="tiny" :disabled="runtimeOpsDisabled" :title="runtimeOpsDisabledReason" :loading="daemonLogsLoading" @click="loadDaemonLogs">刷新</n-button>
          <pre class="daemon-logs">{{ daemonLogs || '（暂无日志，启动守护进程后点击刷新）' }}</pre>
        </n-space>
      </n-space>
    </n-card>

    <n-modal v-model:show="nodeConfigModalVisible" preset="card" title="配置节点" style="width: min(860px, 94vw)">
      <div class="node-config-modal-body">
      <n-space vertical :size="12">
        <p class="status-hint" style="margin: 0">
          先完成网络名、网络密钥与初始节点配置，再进行启动/重启操作。初始节点支持逗号或换行分隔多个地址。
        </p>
        <n-alert type="info" title="节点连接能力">
          当前配置支持同时填写多个初始节点地址（ET_PEERS），系统会按逗号/换行自动识别。
        </n-alert>
        <n-form :model="form" label-placement="left" label-width="120">
          <n-divider style="margin: 0">基础组网</n-divider>
          <n-form-item label="网络名">
            <n-input v-model:value="form.network_name" placeholder="网络标识（ET_NETWORK_NAME）" />
          </n-form-item>
          <n-form-item label="网络密钥">
            <n-input
              v-model:value="form.network_secret"
              type="password"
              show-password-on="click"
              placeholder="网络密钥（ET_NETWORK_SECRET）"
            />
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
          <n-divider style="margin: 0">节点连接</n-divider>
          <n-form-item v-if="networkMode === 'public'" label="公共服务器">
            <n-input v-model:value="publicServer" placeholder="如 tcp://public.easytier.cn:11010" @update:value="(v) => (form.peers = v)" />
            <template #feedback>
              <span class="status-hint">常用格式：`tcp://host:port`，支持 IPv4/域名。</span>
            </template>
          </n-form-item>
          <n-form-item v-else-if="networkMode === 'manual'" label="初始节点（peers）">
            <n-input
              v-model:value="form.peers"
              type="textarea"
              :rows="4"
              placeholder="如 tcp://public.easytier.cn:11010，多个用逗号或换行"
            />
            <template #feedback>
              <span class="status-hint">已识别 {{ peerCount }} 个节点：{{ parsedPeers.slice(0, 3).join('，') || '—' }}<span v-if="peerCount > 3"> ...</span></span>
            </template>
          </n-form-item>
          <n-form-item v-else label="初始节点（peers）">
            <n-input
              type="textarea"
              :rows="2"
              disabled
              placeholder="独立模式下无需填写初始节点。"
            />
          </n-form-item>
          <n-divider style="margin: 0">地址策略</n-divider>
          <n-form-item label="虚拟 IPv4 地址">
            <n-space align="center">
              <n-input
                v-model:value="form.ipv4"
                class="ipv4-input"
                placeholder="如 10.0.111.99"
                :disabled="form.dhcp"
                @update:value="onIPv4Change"
              />
              <span class="cidr-suffix">/24</span>
              <n-checkbox v-model:checked="form.dhcp" @update:checked="onDHCPChange">DHCP</n-checkbox>
            </n-space>
          </n-form-item>
          <n-divider style="margin: 0">高级项（常用）</n-divider>
          <n-form-item label="主机名">
            <n-input v-model:value="form.hostname" placeholder="ET_HOSTNAME（可选）" />
          </n-form-item>
          <n-form-item label="公网发现节点">
            <n-input v-model:value="form.external_node" placeholder="如 tcp://public.easytier.cn:11010（ET_EXTERNAL_NODE）" />
          </n-form-item>
          <n-form-item label="子网代理">
            <n-input
              v-model:value="form.proxy_networks"
              type="textarea"
              :rows="2"
              placeholder="如 10.0.0.0/24，多个子网用逗号或换行（ET_PROXY_NETWORKS）"
            />
          </n-form-item>
          <n-form-item label="RPC 地址">
            <n-input v-model:value="form.rpc_portal" placeholder="如 127.0.0.1:15888（ET_RPC_PORTAL）" />
          </n-form-item>
          <n-form-item label="版本标签">
            <n-input v-model:value="form.image_tag" placeholder="如 latest / v2.2.3（EASYTIER_IMAGE_TAG）" />
          </n-form-item>
          <n-form-item label="启用">
            <n-switch v-model:value="form.enabled" />
          </n-form-item>
        </n-form>
        <n-space justify="end">
          <n-button @click="nodeConfigModalVisible = false">取消</n-button>
          <n-button :loading="saving" @click="onModalSave">保存配置</n-button>
          <n-button type="primary" :loading="saving" @click="onModalSaveAndRestart">保存并重启</n-button>
        </n-space>
      </n-space>
      </div>
    </n-modal>
    <n-modal v-model:show="runtimeModalVisible" preset="card" title="版本与运行时" style="width: min(860px, 94vw)">
      <div class="node-config-modal-body">
        <n-space vertical size="small">
          <n-form-item label="EasyTier 版本">
            <n-select
              v-model:value="selectedVersion"
              :options="releaseOptions"
              placeholder="从 GitHub Releases 选择"
              filterable
              clearable
              style="width: 100%"
              :disabled="!easytierHostSupported"
              @update:value="onVersionOrPlatformChange"
            />
          </n-form-item>
          <n-form-item label="平台（Windows）">
            <n-select
              v-model:value="selectedPlatformKey"
              :options="platformOptions"
              placeholder="Windows amd64 / arm64"
              style="width: 100%"
              :disabled="!easytierHostSupported"
              @update:value="onVersionOrPlatformChange"
            />
          </n-form-item>
          <n-space align="center">
            <n-button :loading="runtimeInstalling" :disabled="!easytierHostSupported" @click="installRuntime">
              下载 EasyTier 运行时
            </n-button>
            <n-button :loading="runtimeRemoving" :disabled="!easytierHostSupported || !runtimeInstalled" @click="removeRuntime">
              移除
            </n-button>
          </n-space>
          <span v-if="runtimeInstalled" class="runtime-hint">已安装：当前所选版本与平台。启动/停止/重启请在上方状态卡片选择已下载版本后操作。<span v-if="!daemonModeEnabled">（需先开启守护进程模式并重启 SkyLink）</span></span>
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
          <n-space justify="end" style="margin-top: 8px">
            <n-button @click="runtimeModalVisible = false">关闭</n-button>
          </n-space>
        </n-space>
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import {
  NCard,
  NForm,
  NFormItem,
  NInput,
  NButton,
  NSpace,
  NDataTable,
  NCollapse,
  NCollapseItem,
  NSwitch,
  NRadioGroup,
  NRadio,
  NCheckbox,
  NDivider,
  NSelect,
  NAlert,
  NModal,
} from 'naive-ui'
import EasyTierIntroAlert from './EasyTierIntroAlert.vue'
import EasyTierProfilesCard from './EasyTierProfilesCard.vue'
import { useEasyTierPage } from '../../composables/easyTier/useEasyTierPage'

const cliRawTargetOptions = [
  { label: 'peer（对等节点）', value: 'peer' },
  { label: 'route（路由）', value: 'route' },
  { label: 'node（本机）', value: 'node' },
  { label: 'version（版本）', value: 'version' },
]
const nodeConfigModalVisible = ref(false)
const runtimeModalVisible = ref(false)
function openNodeConfigModal() {
  nodeConfigModalVisible.value = true
}
function openRuntimeModal() {
  runtimeModalVisible.value = true
}

const {
  form,
  daemonStatus,
  daemonModeEnabled,
  networkMode,
  publicServer,
  saving,
  statusLoading,
  versionCheckLoading,
  lastStatusUpdated,
  status,
  versionCheck,
  easytierHostSupported,
  releasesError,
  releaseOptions,
  platformOptions,
  selectedVersion,
  selectedPlatformKey,
  runtimeInstalling,
  runtimeRemoving,
  runtimeInstalled,
  platformError,
  runtimeError,
  installedList,
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
  parsedCliPeerRows,
  cliPeerSummary,
  cliPeerColumns,
  activeProfileId,
  newProfileName,
  profileOptions,
  canDeleteProfile,
  parsedPeers,
  peerCount,
  hasNodeConfig,
  runtimeOpsDisabled,
  runtimeOpsDisabledReason,
  aggregatedErrors,
  installedOptionsForStatus,
  selectedInstalledKey,
  displayNodes,
  peerResidentRows,
  peerResidentColumns,
  nodeTableColumns,
  routeColumns,
  installedListColumns,
  onDHCPChange,
  onIPv4Change,
  onProfileChange,
  createProfile,
  deleteCurrentProfile,
  onVersionOrPlatformChange,
  saveByModal,
  saveAndRestartByModal,
  loadStatus,
  loadDaemonLogs,
  stopDaemon,
  startOrRestartDaemon,
  releasePort,
  installRuntime,
  removeRuntime,
  checkUpdate,
  useLatestVersion,
} = useEasyTierPage()

async function onModalSave() {
  const ok = await saveByModal()
  if (ok) {
    nodeConfigModalVisible.value = false
  }
}

async function onModalSaveAndRestart() {
  const ok = await saveAndRestartByModal()
  if (ok) {
    nodeConfigModalVisible.value = false
  }
}

</script>

<style scoped>
.easytier-panel {
  padding-bottom: 8px;
}
.easytier-tab-lead {
  margin: 0 0 12px 0;
  font-size: 13px;
  color: #64748b;
  line-height: 1.5;
}
.panel-section {
  margin-bottom: 16px;
}
.easytier-error-list {
  margin: 0;
  padding-left: 1.2em;
}
.easytier-version-hint {
  font-size: 12px;
  color: #64748b;
}
.easytier-version-hint a {
  margin-left: 8px;
}
.status-version,
.status-self {
  font-size: 13px;
  color: #64748b;
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
  color: #64748b;
}
.status-hint code {
  font-family: var(--n-font-mono, monospace);
  font-size: 12px;
  padding: 0 4px;
  background: #f8fafc;
  border-radius: 3px;
  border: 1px solid #e2e8f0;
}
.status-updated {
  margin-left: 12px;
  font-size: 12px;
  color: #94a3b8;
}
.runtime-hint {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: #64748b;
}
.daemon-logs {
  margin: 0;
  padding: 10px;
  max-height: 280px;
  overflow: auto;
  font-size: 12px;
  line-height: 1.4;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  white-space: pre-wrap;
  word-break: break-all;
}
.cli-raw-label {
  margin-top: 10px;
  font-size: 12px;
  color: #94a3b8;
}
.cli-raw-block {
  margin: 4px 0 0 0;
  padding: 10px;
  max-height: 320px;
  overflow: auto;
  font-size: 12px;
  line-height: 1.4;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  white-space: pre-wrap;
  word-break: break-all;
}
.cli-raw-err {
  background: #fff1f2;
  border-color: #fecdd3;
}
.peer-modern-view {
  margin-top: 12px;
}
.peer-modern-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 10px;
  margin-bottom: 10px;
}
.peer-stat-card {
  padding: 10px 12px;
  border-radius: 10px;
  background: #ffffff;
  border: 1px solid #e2e8f0;
}
.peer-stat-label {
  font-size: 12px;
  color: #64748b;
}
.peer-stat-value {
  margin-top: 4px;
  font-size: 18px;
  font-weight: 600;
  color: #0f172a;
}
.peer-stat-success {
  color: #059669;
}
.peer-stat-warn {
  color: #d97706;
}
.node-config-modal-body {
  max-height: 75vh;
  overflow-y: auto;
  padding-right: 4px;
}
</style>
