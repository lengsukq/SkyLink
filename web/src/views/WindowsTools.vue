<template>
  <div>
    <page-header
      title="Windows 工具"
      description="统一管理 Windows 专用能力（SMB / WebDAV）。"
    />

    <n-card class="page-section page-card">
      <n-tabs v-model:value="activeTab" type="line" animated>
        <n-tab-pane name="smb" tab="SMB">
          <smb-view />
        </n-tab-pane>
        <n-tab-pane name="webdav" tab="WebDAV">
          <web-dev-view />
        </n-tab-pane>
      </n-tabs>
    </n-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { NCard, NTabs, NTabPane } from 'naive-ui'
import PageHeader from '../components/PageHeader.vue'
import SmbView from './Smb.vue'
import WebDevView from './WebDev.vue'

const activeTab = ref('smb')
</script>
