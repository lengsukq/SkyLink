<template>
  <n-modal :show="show" preset="card" title="版本与运行时" style="width: min(860px, 94vw)" @update:show="emit('update:show', $event)">
    <div class="node-config-modal-body">
      <n-space vertical size="small">
        <n-form-item label="EasyTier 版本">
          <n-select
            :value="selectedVersion"
            :options="releaseOptions"
            placeholder="从 GitHub Releases 选择"
            filterable
            clearable
            style="width: 100%"
            :disabled="!easytierHostSupported"
            @update:value="onVersionChange"
          />
        </n-form-item>
        <n-form-item label="平台（Windows）">
          <n-select
            :value="selectedPlatformKey"
            :options="platformOptions"
            placeholder="Windows amd64 / arm64"
            style="width: 100%"
            :disabled="!easytierHostSupported"
            @update:value="onPlatformChange"
          />
        </n-form-item>
        <n-space align="center">
          <n-button :loading="runtimeInstalling" :disabled="!easytierHostSupported" @click="emit('installRuntime')">
            下载 EasyTier 运行时
          </n-button>
          <n-button :loading="runtimeRemoving" :disabled="!easytierHostSupported || !runtimeInstalled" @click="emit('removeRuntime')">
            移除
          </n-button>
        </n-space>
        <span v-if="runtimeInstalled" class="runtime-hint">已安装：当前所选版本与平台。启动/停止/重启请在上方状态卡片选择已下载版本后操作。<span v-if="!daemonModeEnabled">（需先开启守护进程模式并重启 SkyLink）</span></span>
        <span v-else-if="selectedVersion && selectedPlatformKey" class="runtime-hint">未安装</span>
        <span v-if="releasesError" class="status-error">{{ releasesError }}</span>
        <span v-if="runtimeError" class="status-error">{{ runtimeError }}</span>
        <span v-if="platformError" class="status-error">{{ platformError }}</span>
        <n-space align="center" style="margin-top: 8px">
          <n-button :loading="versionCheckLoading" size="tiny" @click="emit('checkUpdate')">检查更新</n-button>
          <span v-if="versionCheck.latest_version" class="easytier-version-hint">
            最新: {{ versionCheck.latest_version }}
            <a v-if="versionCheck.release_url" :href="versionCheck.release_url" target="_blank" rel="noopener">Release</a>
            <n-button v-if="versionCheck.update_available" text type="primary" size="tiny" @click="emit('useLatestVersion')">使用此版本</n-button>
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
          <n-button @click="emit('update:show', false)">关闭</n-button>
        </n-space>
      </n-space>
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import { NModal, NSpace, NFormItem, NSelect, NButton, NDivider, NDataTable, type DataTableColumns, type SelectOption } from 'naive-ui'
import type { RuntimeInstalledItem, EasyTierVersionCheckResponse } from '../../composables/easyTier/types'

const props = defineProps<{
  show: boolean
  easytierHostSupported: boolean
  releaseOptions: SelectOption[]
  platformOptions: SelectOption[]
  selectedVersion: string | null
  selectedPlatformKey: string | null
  runtimeInstalling: boolean
  runtimeRemoving: boolean
  runtimeInstalled: boolean
  daemonModeEnabled: boolean
  releasesError: string
  runtimeError: string
  platformError: string
  versionCheckLoading: boolean
  versionCheck: EasyTierVersionCheckResponse
  installedList: RuntimeInstalledItem[]
  installedListColumns: DataTableColumns<RuntimeInstalledItem>
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
  'update:selectedVersion': [value: string | null]
  'update:selectedPlatformKey': [value: string | null]
  onVersionOrPlatformChange: []
  installRuntime: []
  removeRuntime: []
  checkUpdate: []
  useLatestVersion: []
}>()

function onVersionChange(value: string | null) {
  emit('update:selectedVersion', value)
  emit('onVersionOrPlatformChange')
}

function onPlatformChange(value: string | null) {
  emit('update:selectedPlatformKey', value)
  emit('onVersionOrPlatformChange')
}
</script>

<style scoped>
.node-config-modal-body {
  max-height: 75vh;
  overflow-y: auto;
  padding-right: 4px;
}
.runtime-hint {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: #64748b;
}
.status-error {
  color: var(--n-error-color);
  font-size: 13px;
}
.easytier-version-hint {
  font-size: 12px;
  color: #64748b;
}
.easytier-version-hint a {
  margin-left: 8px;
}
</style>
