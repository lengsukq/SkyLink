<template>
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
        <n-button text type="primary" style="padding: 0 2px" @click="emit('openNodeConfigModal')">
          配置节点
        </n-button>
        ，完成网络名、密钥和初始节点后再启动/重启。
      </n-alert>
      <n-space align="center" flex-wrap="wrap">
        <n-button size="small" :loading="statusLoading" :disabled="runtimeOpsDisabled" :title="runtimeOpsDisabledReason" @click="emit('loadStatus')">刷新</n-button>
        <n-button
          size="small"
          type="primary"
          ghost
          @click="emit('openNodeConfigModal')"
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
          :value="selectedInstalledKey"
          :options="installedOptionsForStatus"
          placeholder="选择要启动的版本"
          clearable
          style="min-width: 180px"
          size="small"
          @update:value="emit('update:selectedInstalledKey', $event)"
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
          @click="emit('stopDaemon')"
        >停止</n-button>
        <n-button
          size="tiny"
          :disabled="runtimeOpsDisabled"
          :title="runtimeOpsDisabled ? runtimeOpsDisabledReason : (daemonStatus.running ? '重启 EasyTier 守护进程' : '启动 EasyTier 守护进程')"
          @click="emit('startOrRestartDaemon')"
        >{{ daemonStatus.running ? '重启' : '启动' }}</n-button>
        <n-button
          size="tiny"
          :disabled="runtimeOpsDisabled"
          :loading="releasePortLoading"
          :title="runtimeOpsDisabled ? runtimeOpsDisabledReason : '结束占用 EasyTier 相关端口的进程（RPC 15888、listeners 11010–11013 等），便于重新启动'"
          @click="emit('releasePort')"
        >解除端口占用</n-button>
        <span v-if="!daemonModeEnabled" class="status-hint">
          请在配置中开启 <code>easytier.daemon_enabled</code> 并重启 SkyLink 后，即可在此处控制守护进程。
        </span>
      </n-space>
      <div v-if="!formEnabled" class="status-hint">
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
          :value="cliRawTarget"
          :options="cliRawTargetOptions"
          style="min-width: 180px"
          size="small"
          :disabled="!easytierHostSupported"
          @update:value="emit('update:cliRawTarget', $event)"
        />
        <n-button size="small" :loading="cliRawLoading" :disabled="!easytierHostSupported" @click="emit('loadCliRaw')">获取</n-button>
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
        <n-button size="tiny" :disabled="runtimeOpsDisabled" :title="runtimeOpsDisabledReason" :loading="daemonLogsLoading" @click="emit('loadDaemonLogs')">刷新</n-button>
        <pre class="daemon-logs">{{ daemonLogs || '（暂无日志，启动守护进程后点击刷新）' }}</pre>
      </n-space>
    </n-space>
  </n-card>
</template>

<script setup lang="ts">
import {
  NCard,
  NButton,
  NSpace,
  NDataTable,
  NCollapse,
  NCollapseItem,
  NDivider,
  NSelect,
  NAlert,
  type DataTableColumns,
  type SelectOption,
} from 'naive-ui'
import type { CliPeerRow, CliPeerSummary } from '../../composables/easyTier/cliRawParser'
import type { DisplayNodeRow, EasyTierStatusRoute, EasyTierStatusState } from '../../composables/easyTier/types'

defineProps<{
  aggregatedErrors: string[]
  hasNodeConfig: boolean
  statusLoading: boolean
  runtimeOpsDisabled: boolean
  runtimeOpsDisabledReason: string
  daemonStatus: { started_version: string; running: boolean; pid: number }
  status: EasyTierStatusState
  lastStatusUpdated: string
  installedOptionsForStatus: SelectOption[]
  selectedInstalledKey: string | null
  daemonModeEnabled: boolean
  releasePortLoading: boolean
  formEnabled: boolean
  displayNodes: DisplayNodeRow[]
  nodeTableColumns: DataTableColumns<DisplayNodeRow>
  peerResidentRows: CliPeerRow[]
  peerResidentColumns: DataTableColumns<CliPeerRow>
  routeColumns: DataTableColumns<EasyTierStatusRoute>
  cliRawTarget: 'peer' | 'route' | 'node' | 'version'
  cliRawTargetOptions: SelectOption[]
  easytierHostSupported: boolean
  cliRawLoading: boolean
  cliRawMetaError: string
  parsedCliPeerRows: CliPeerRow[]
  cliPeerSummary: CliPeerSummary
  cliPeerColumns: DataTableColumns<CliPeerRow>
  cliRawStdout: string
  cliRawStderr: string
  daemonLogsLoading: boolean
  daemonLogs: string
}>()

const emit = defineEmits<{
  openNodeConfigModal: []
  loadStatus: []
  'update:selectedInstalledKey': [value: string | null]
  stopDaemon: []
  startOrRestartDaemon: []
  releasePort: []
  'update:cliRawTarget': [value: 'peer' | 'route' | 'node' | 'version']
  loadCliRaw: []
  loadDaemonLogs: []
}>()
</script>

<style scoped>
.panel-section {
  margin-bottom: 16px;
}
.easytier-error-list {
  margin: 0;
  padding-left: 1.2em;
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
</style>
