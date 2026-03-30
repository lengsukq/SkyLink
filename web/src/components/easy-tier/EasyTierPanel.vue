<template>
  <div class="easytier-panel">
    <p class="easytier-tab-lead">
      Mesh VPN 与运行时管理；映射时可选用 mesh 内节点作为后端。完整集成仅在 <strong>Windows</strong> 版 SkyLink 上可用。
    </p>

    <easy-tier-top-cards
      :easytier-host-supported="easytierHostSupported"
      :easytier-autostart="easytierAutostart"
      :easytier-autostart-saving="easytierAutostartSaving"
      :active-profile-id="activeProfileId"
      :new-profile-name="newProfileName"
      :profile-options="profileOptions"
      :can-delete="canDeleteProfile"
      @toggle-autostart="onToggleEasyTierAutostart"
      @update:active-profile-id="activeProfileId = $event"
      @update:new-profile-name="newProfileName = $event"
      @profile-change="onProfileChange"
      @create-profile="createProfile"
      @delete-profile="deleteCurrentProfile"
      @open-runtime="openRuntimeModal"
    />

    <easy-tier-status-card
      :aggregated-errors="aggregatedErrors"
      :has-node-config="hasNodeConfig"
      :status-loading="statusLoading"
      :runtime-ops-disabled="runtimeOpsDisabled"
      :runtime-ops-disabled-reason="runtimeOpsDisabledReason"
      :daemon-status="daemonStatus"
      :status="status"
      :last-status-updated="lastStatusUpdated"
      :installed-options-for-status="installedOptionsForStatus"
      :selected-installed-key="selectedInstalledKey"
      :daemon-mode-enabled="daemonModeEnabled"
      :release-port-loading="releasePortLoading"
      :form-enabled="form.enabled"
      :display-nodes="displayNodes"
      :node-table-columns="nodeTableColumns"
      :peer-resident-rows="peerResidentRows"
      :peer-resident-columns="peerResidentColumns"
      :route-columns="routeColumns"
      :cli-raw-target="cliRawTarget"
      :cli-raw-target-options="CLI_RAW_TARGET_OPTIONS"
      :easytier-host-supported="easytierHostSupported"
      :cli-raw-loading="cliRawLoading"
      :cli-raw-meta-error="cliRawMetaError"
      :parsed-cli-peer-rows="parsedCliPeerRows"
      :cli-peer-summary="cliPeerSummary"
      :cli-peer-columns="cliPeerColumns"
      :cli-raw-stdout="cliRawStdout"
      :cli-raw-stderr="cliRawStderr"
      :daemon-logs-loading="daemonLogsLoading"
      :daemon-logs="daemonLogs"
      @open-node-config-modal="openNodeConfigModal"
      @load-status="loadStatus"
      @update:selected-installed-key="selectedInstalledKey = $event"
      @stop-daemon="stopDaemon"
      @start-or-restart-daemon="startOrRestartDaemon"
      @release-port="releasePort"
      @update:cli-raw-target="cliRawTarget = $event"
      @load-cli-raw="loadCliRaw"
      @load-daemon-logs="loadDaemonLogs"
    />

    <easy-tier-node-config-modal
      :show="nodeConfigModalVisible"
      :form="form"
      :network-mode="networkMode"
      :public-server="publicServer"
      :peer-count="peerCount"
      :parsed-peers="parsedPeers"
      :saving="saving"
      :on-dhcp-change="onDHCPChange"
      :on-ipv4-change="onIPv4Change"
      @update:show="nodeConfigModalVisible = $event"
      @update:network-mode="networkMode = $event"
      @update:public-server="publicServer = $event"
      @save="onModalSave"
      @save-and-restart="onModalSaveAndRestart"
    />
    <easy-tier-runtime-modal
      :show="runtimeModalVisible"
      :easytier-host-supported="easytierHostSupported"
      :release-options="releaseOptions"
      :platform-options="platformOptions"
      :selected-version="selectedVersion"
      :selected-platform-key="selectedPlatformKey"
      :runtime-installing="runtimeInstalling"
      :runtime-removing="runtimeRemoving"
      :runtime-installed="runtimeInstalled"
      :daemon-mode-enabled="daemonModeEnabled"
      :releases-error="releasesError"
      :runtime-error="runtimeError"
      :platform-error="platformError"
      :version-check-loading="versionCheckLoading"
      :version-check="versionCheck"
      :installed-list="installedList"
      :installed-list-columns="installedListColumns"
      @update:show="runtimeModalVisible = $event"
      @update:selected-version="selectedVersion = $event"
      @update:selected-platform-key="selectedPlatformKey = $event"
      @on-version-or-platform-change="onVersionOrPlatformChange"
      @install-runtime="installRuntime"
      @remove-runtime="removeRuntime"
      @check-update="checkUpdate"
      @use-latest-version="useLatestVersion"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import EasyTierNodeConfigModal from './EasyTierNodeConfigModal.vue'
import EasyTierRuntimeModal from './EasyTierRuntimeModal.vue'
import EasyTierStatusCard from './EasyTierStatusCard.vue'
import EasyTierTopCards from './EasyTierTopCards.vue'
import { useEasyTierPage } from '../../composables/easyTier/useEasyTierPage'
import { CLI_RAW_TARGET_OPTIONS } from '../../composables/easyTier/constants'

const nodeConfigModalVisible = ref(false)
const runtimeModalVisible = ref(false)
function openNodeConfigModal() {
  nodeConfigModalVisible.value = true
}
function openRuntimeModal() {
  runtimeModalVisible.value = true
}

const page = useEasyTierPage()

const { easytierHostSupported } = page.meta

const {
  activeProfileId,
  newProfileName,
  profileOptions,
  canDeleteProfile,
  onProfileChange,
  createProfile,
  deleteCurrentProfile,
} = page.profile

const {
  form,
  networkMode,
  publicServer,
  saving,
  parsedPeers,
  peerCount,
  onDHCPChange,
  onIPv4Change,
  saveByModal,
  saveAndRestartByModal,
} = page.config

const {
  statusLoading,
  lastStatusUpdated,
  status,
  aggregatedErrors,
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
  hasNodeConfig,
} = page.statusView

const {
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
  installedOptionsForStatus,
  selectedInstalledKey,
  installedListColumns,
  onVersionOrPlatformChange,
  installRuntime,
  removeRuntime,
  versionCheckLoading,
  versionCheck,
  checkUpdate,
  useLatestVersion,
} = page.runtime

const {
  daemonStatus,
  daemonModeEnabled,
  daemonLogs,
  daemonLogsLoading,
  releasePortLoading,
  easytierAutostart,
  easytierAutostartSaving,
  onToggleEasyTierAutostart,
  loadDaemonLogs,
  stopDaemon,
  startOrRestartDaemon,
  releasePort,
} = page.daemon

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
</style>
