<template>
  <div class="easytier-panel">
    <p class="easytier-tab-lead">
      Mesh VPN 与运行时管理；映射时可选用 mesh 内节点作为后端。完整集成仅在 <strong>Windows</strong> 版 SkyLink 上可用。
    </p>

    <n-alert v-if="!easytierHostSupported" type="warning" title="当前环境不支持 EasyTier 集成" class="panel-section">
      SkyLink 内置的 EasyTier（守护进程、下载 Windows 运行时、端口释放与 easytier-cli 状态）仅在 <strong>Windows 主机</strong>上可用。
      若你使用 Linux/Docker 等部署，本标签无法保存配置或拉起进程；请在 Windows 上运行 SkyLink，或自行在其他环境部署 EasyTier。
    </n-alert>

    <n-card v-if="easytierHostSupported" size="small" title="SkyLink 启动选项" class="panel-section page-card">
      <n-form label-placement="left" label-width="220">
        <n-form-item label="SkyLink 启动时自动启动 EasyTier">
          <n-switch
            :value="easytierAutostart"
            :loading="easytierAutostartSaving"
            @update:value="onToggleEasyTierAutostart"
          />
        </n-form-item>
      </n-form>
      <p class="status-hint">
        需已配置网络名、网络密钥和初始节点（peers）并开启下方「启用」。仅影响进程随 SkyLink 自启，本标签内手动启动/重启不受影响。
      </p>
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
            :disabled="!easytierHostSupported || !daemonModeEnabled || !daemonStatus.running"
            :title="!easytierHostSupported ? '仅在 Windows 上可用' : !daemonModeEnabled ? '需在配置中开启 easytier.daemon_enabled 并重启 SkyLink' : (!daemonStatus.running ? '守护进程未运行' : '')"
            @click="stopDaemon"
          >停止</n-button>
          <n-button
            size="tiny"
            :disabled="!easytierHostSupported || !daemonModeEnabled"
            :title="!easytierHostSupported ? '仅在 Windows 上可用' : daemonModeEnabled ? (daemonStatus.running ? '重启 EasyTier 守护进程' : '启动 EasyTier 守护进程') : '需在配置中开启 easytier.daemon_enabled 并重启 SkyLink'"
            @click="startOrRestartDaemon"
          >{{ daemonStatus.running ? '重启' : '启动' }}</n-button>
          <n-button
            size="tiny"
            :disabled="!easytierHostSupported"
            :loading="releasePortLoading"
            :title="!easytierHostSupported ? '仅在 Windows 上可用' : '结束占用 EasyTier 相关端口的进程（RPC 15888、listeners 11010–11013 等），便于重新启动'"
            @click="releasePort"
          >解除端口占用</n-button>
          <span v-if="!daemonModeEnabled" class="status-hint">
            请在配置中开启 <code>easytier.daemon_enabled</code> 并重启 SkyLink 后，即可在此处控制守护进程。
          </span>
        </n-space>
        <n-divider v-if="daemonModeEnabled" style="margin: 12px 0 8px 0">守护进程日志</n-divider>
        <p v-if="daemonModeEnabled && easytierHostSupported" class="status-hint" style="margin: 0 0 8px 0">
          默认在弹出的独立控制台中查看 easytier 节点界面；此处仅在有日志时显示。若需把输出同步到本页，请设置环境变量
          <code>SKYLINK_EASYTIER_DAEMON_CAPTURE_OUTPUT=1</code> 并重启 SkyLink。
        </p>
        <n-space v-if="daemonModeEnabled" vertical size="small">
          <n-button size="tiny" :disabled="!easytierHostSupported" :loading="daemonLogsLoading" @click="loadDaemonLogs">刷新</n-button>
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

    <n-card title="在网信息（easytier-cli 原始输出）" class="panel-section page-card">
      <p class="status-hint" style="margin-bottom: 8px">
        对应文档中的管理命令输出；用于核对 peers、路由与本机节点。
        <a href="https://easytier.cn/guide/network/configurations.html" target="_blank" rel="noopener noreferrer">完整配置选项</a>
        中的环境变量（如 <code>ET_RPC_PORTAL</code>）需与下方「高级」中的 RPC 一致。
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
      <template v-if="cliRawStdout || cliRawStderr">
        <div v-if="cliRawStdout" class="cli-raw-label">stdout</div>
        <pre v-if="cliRawStdout" class="cli-raw-block">{{ cliRawStdout }}</pre>
        <div v-if="cliRawStderr" class="cli-raw-label">stderr</div>
        <pre v-if="cliRawStderr" class="cli-raw-block cli-raw-err">{{ cliRawStderr }}</pre>
      </template>
      <p v-else-if="easytierHostSupported && !cliRawLoading && !cliRawMetaError" class="status-hint" style="margin-top: 8px">
        选择子命令后点击「获取」。需守护进程已运行；CLI 与 core 同目录（推荐）或可在 PATH 中找到 <code>easytier-cli</code>。
      </p>
    </n-card>

    <n-grid cols="1 s:1 m:2 l:2 xl:3" x-gap="16" y-gap="16" class="panel-section">
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
          </n-space>
        </n-card>
      </n-gi>

      <n-gi>
        <n-card title="网络配置" class="page-card">
          <div class="status-hint" style="margin-bottom: 8px">
            启用后，可通过上方状态卡片控制 EasyTier 守护进程运行，并查看 mesh 状态。
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
                placeholder="如 10.0.0.0/24，多个子网用逗号或换行（ET_PROXY_NETWORKS / proxy_networks）"
              />
            </n-form-item>
            <n-form-item label="启用">
              <n-switch v-model:value="form.enabled" />
            </n-form-item>
            <n-form-item v-if="showAdvanced" label="RPC 地址">
              <n-input v-model:value="form.rpc_portal" placeholder="SkyLink 连接 EasyTier 的地址，如 easytier:15888" />
            </n-form-item>
          </n-form>
          <n-space>
            <n-button type="primary" :loading="saving" :disabled="!easytierHostSupported" @click="save">保存配置</n-button>
            <n-button :loading="saving" :disabled="!easytierHostSupported" @click="saveAndRestart">保存并重启</n-button>
            <n-button @click="showAdvanced = !showAdvanced">{{ showAdvanced ? '收起高级' : '高级' }}</n-button>
          </n-space>
        </n-card>
      </n-gi>
    </n-grid>
  </div>
</template>

<script setup lang="ts">
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
  NGrid,
  NGi,
  NSelect,
  NAlert,
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

const {
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
  onProfileChange,
  createProfile,
  deleteCurrentProfile,
  onVersionOrPlatformChange,
  save,
  saveAndRestart,
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
  background: var(--n-color-target, #f0f0f0);
  border-radius: 3px;
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
  background: var(--n-color-target, #f5f5f5);
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
  background: var(--n-color-target, #f5f5f5);
  border-radius: 6px;
  white-space: pre-wrap;
  word-break: break-all;
}
.cli-raw-err {
  background: rgba(239, 68, 68, 0.08);
}
</style>
