<template>
  <div>
    <page-header title="个人网盘" description="文件浏览、上传下载、类型分类筛选（支持递归搜索）。" />
    <n-card class="page-section page-card">
      <drive-user-browser-panel />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { NCard } from 'naive-ui'
import PageHeader from '../components/PageHeader.vue'
import DriveUserBrowserPanel from '../components/drive/DriveUserBrowserPanel.vue'
</script>

