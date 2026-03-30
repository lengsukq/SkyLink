<template>
  <div class="drive-page">
    <n-card class="page-section page-card">
      <drive-user-browser-panel />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { NCard } from 'naive-ui'
import DriveUserBrowserPanel from '../components/drive/DriveUserBrowserPanel.vue'
</script>

<style scoped>
.drive-page {
  padding-top: 4px;
}
</style>

