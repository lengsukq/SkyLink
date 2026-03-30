<template>
  <div>
    <page-header
      title="Windows 工具"
      description="Windows 专用能力：本机磁盘概况、SMB 共享、个人网盘管理，以及 EasyTier mesh VPN（仅 Windows 主机完整支持）。"
    />

    <n-alert type="info" class="page-section">
      跨平台 WebDAV 请在
      <router-link to="/file-services">文件服务</router-link>
      中配置。
    </n-alert>

    <storage-volumes-panel class="page-section" />

    <n-card class="page-section page-card">
      <n-tabs v-model:value="activeTab" type="line" animated>
        <n-tab-pane name="smb" tab="SMB">
          <smb-view />
        </n-tab-pane>
        <n-tab-pane name="drive" tab="个人网盘">
          <personal-drive-view />
        </n-tab-pane>
        <n-tab-pane name="easytier" tab="EasyTier">
          <easy-tier-panel />
        </n-tab-pane>
      </n-tabs>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NAlert, NCard, NTabs, NTabPane } from 'naive-ui'
import PageHeader from '../components/PageHeader.vue'
import StorageVolumesPanel from '../components/StorageVolumesPanel.vue'
import EasyTierPanel from '../components/easy-tier/EasyTierPanel.vue'
import SmbView from './Smb.vue'
import PersonalDriveView from './PersonalDrive.vue'

const route = useRoute()
const router = useRouter()

const validTabs = ['smb', 'drive', 'easytier'] as const
type ToolTab = (typeof validTabs)[number]

const activeTab = ref<ToolTab>('smb')

function isToolTab(v: string): v is ToolTab {
  return (validTabs as readonly string[]).includes(v)
}

function readTabFromRoute(): ToolTab {
  const raw = route.query.tab
  const t = typeof raw === 'string' ? raw : ''
  return isToolTab(t) ? t : 'smb'
}

onMounted(() => {
  activeTab.value = readTabFromRoute()
})

watch(
  () => route.query.tab,
  () => {
    const next = readTabFromRoute()
    if (next !== activeTab.value) activeTab.value = next
  },
)

watch(activeTab, (tab) => {
  if (route.query.tab === tab) return
  router.replace({ query: { ...route.query, tab } })
})
</script>
<｜tool▁calls▁begin｜><｜tool▁call▁begin｜>
StrReplace