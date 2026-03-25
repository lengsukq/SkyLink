<template>
  <div>
    <page-header title="个人网盘" description="Windows 专属：管理员创建账号并分配本机目录/空间；账号通过用户名+密码登录个人网盘。" />

    <n-card class="page-section page-card">
      <drive-accounts-panel />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { NCard } from 'naive-ui'
import PageHeader from '../components/PageHeader.vue'
import DriveAccountsPanel from '../components/drive/DriveAccountsPanel.vue'
</script>

