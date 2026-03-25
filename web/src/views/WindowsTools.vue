<template>
  <div>
    <page-header
      title="Windows 工具"
      description="Windows 专用能力：本机磁盘概况与 SMB 共享。"
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
      </n-tabs>
    </n-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { NAlert, NCard, NTabs, NTabPane } from 'naive-ui'
import PageHeader from '../components/PageHeader.vue'
import StorageVolumesPanel from '../components/StorageVolumesPanel.vue'
import SmbView from './Smb.vue'
import PersonalDriveView from './PersonalDrive.vue'

const activeTab = ref('smb')
</script>
