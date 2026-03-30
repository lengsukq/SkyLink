<template>
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
        @update:value="emit('toggleAutostart', $event)"
      />
    </n-space>
  </n-card>

  <easy-tier-intro-alert />

  <easy-tier-profiles-card
    :active-profile-id="activeProfileId"
    :new-profile-name="newProfileName"
    :profile-options="profileOptions"
    :can-delete="canDelete"
    @update:active-profile-id="emit('update:activeProfileId', $event)"
    @update:new-profile-name="emit('update:newProfileName', $event)"
    @profile-change="emit('profileChange', $event)"
    @create="emit('createProfile')"
    @delete="emit('deleteProfile')"
    @open-runtime="emit('openRuntime')"
  />
</template>

<script setup lang="ts">
import { NAlert, NCard, NSpace, NSwitch } from 'naive-ui'
import EasyTierIntroAlert from './EasyTierIntroAlert.vue'
import EasyTierProfilesCard from './EasyTierProfilesCard.vue'

defineProps<{
  easytierHostSupported: boolean
  easytierAutostart: boolean
  easytierAutostartSaving: boolean
  activeProfileId: string
  newProfileName: string
  profileOptions: { label: string; value: string }[]
  canDelete: boolean
}>()

const emit = defineEmits<{
  toggleAutostart: [value: boolean]
  'update:activeProfileId': [value: string]
  'update:newProfileName': [value: string]
  profileChange: [value: string]
  createProfile: []
  deleteProfile: []
  openRuntime: []
}>()
</script>

<style scoped>
.panel-section {
  margin-bottom: 16px;
}
</style>
