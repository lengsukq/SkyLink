<template>
  <div>
    <page-header
      title="个人网盘"
      description="Windows 专属：管理员创建账号并分配本机目录/空间；账号通过用户名+密码登录个人网盘。"
    />

    <n-alert type="info" show-icon class="page-section drive-links-alert">
      <template #header>用户访问链接</template>
      <div class="drive-links">
        <p class="drive-links__hint">将链接发给用户即可；点击链接会在<strong>新标签页</strong>打开。</p>
        <div class="drive-links__row">
          <span class="drive-links__label">登录页</span>
          <a class="drive-links__url" :href="driveLoginUrl" target="_blank" rel="noopener noreferrer">{{ driveLoginUrl }}</a>
          <n-button size="tiny" quaternary @click="copyUrl(driveLoginUrl)">复制</n-button>
        </div>
        <div class="drive-links__row">
          <span class="drive-links__label">文件门户</span>
          <a class="drive-links__url" :href="drivePortalUrl" target="_blank" rel="noopener noreferrer">{{ drivePortalUrl }}</a>
          <n-button size="tiny" quaternary @click="copyUrl(drivePortalUrl)">复制</n-button>
        </div>
        <p class="drive-links__sub">文件门户需已登录网盘；未登录会自动进入登录页。</p>
      </div>
    </n-alert>

    <n-card class="page-section page-card">
      <drive-accounts-panel />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NAlert, NButton, NCard } from 'naive-ui'
import PageHeader from '../components/PageHeader.vue'
import DriveAccountsPanel from '../components/drive/DriveAccountsPanel.vue'
import { ROUTE_PATHS } from '../constants/routes'
import { copyToClipboard } from '../utils/clipboard'
import { notifyError, notifySuccess } from '../ui/notify'

function hashAppUrl(routePath: string): string {
  const u = new URL(window.location.href)
  const path = routePath.startsWith('/') ? routePath : `/${routePath}`
  u.hash = `#${path}`
  return u.toString()
}

const driveLoginUrl = computed(() => hashAppUrl(ROUTE_PATHS.driveLogin))
const drivePortalUrl = computed(() => hashAppUrl(ROUTE_PATHS.drivePortal))

async function copyUrl(s: string) {
  try {
    await copyToClipboard(s)
    notifySuccess('已复制', '链接已复制到剪贴板')
  } catch (e: any) {
    notifyError('复制失败', e?.message || String(e))
  }
}
</script>

<style scoped>
.drive-links-alert {
  max-width: 960px;
}
.drive-links {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.drive-links__hint {
  margin: 0 0 4px;
  font-size: 13px;
  line-height: 1.5;
  color: var(--n-text-color-3);
}
.drive-links__row {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 8px 10px;
}
.drive-links__label {
  flex: 0 0 auto;
  font-weight: 600;
  font-size: 13px;
  min-width: 4em;
}
.drive-links__sub {
  margin: 0;
  padding-left: calc(4em + 10px);
  font-size: 12px;
  color: var(--n-text-color-3);
  line-height: 1.4;
}
@media (max-width: 559px) {
  .drive-links__sub {
    padding-left: 0;
  }
}
.drive-links__url {
  flex: 1 1 200px;
  min-width: 0;
  font-size: 13px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  word-break: break-all;
  color: var(--n-primary-color);
  text-decoration: underline;
  text-underline-offset: 2px;
}
.drive-links__url:hover {
  opacity: 0.88;
}
</style>
