<template>
  <n-config-provider :theme-overrides="themeOverrides">
    <n-notification-provider>
      <notifier-bridge />
      <n-message-provider>
        <n-layout class="app-layout">
          <n-layout-header class="app-header" bordered>
            <div class="app-header__inner">
              <div class="app-header__left">
                <span class="app-header__logo-text">SkyLink</span>
                <n-space align="center" :size="12">
                  <n-button
                    v-for="item in navItems"
                    :key="item.path"
                    quaternary
                    :type="isActive(item.path) ? 'primary' : 'default'"
                    size="small"
                    @click="go(item.path)"
                  >
                    {{ item.label }}
                  </n-button>
                </n-space>
              </div>
              <div class="app-header__right">
                <n-button
                  quaternary
                  tag="a"
                  href="https://github.com/lengsukq/SkyLink"
                  target="_blank"
                  rel="noopener noreferrer"
                  size="small"
                >
                  GitHub
                </n-button>
              </div>
            </div>
          </n-layout-header>
          <n-layout-content class="app-content">
            <div class="app-content__inner">
              <router-view />
            </div>
          </n-layout-content>
        </n-layout>
      </n-message-provider>
    </n-notification-provider>
  </n-config-provider>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  NConfigProvider,
  NLayout,
  NLayoutHeader,
  NLayoutContent,
  NSpace,
  NButton,
  NMessageProvider,
  NNotificationProvider,
} from 'naive-ui'
import NotifierBridge from './components/NotifierBridge.vue'

const themeOverrides = {
  common: {
    borderRadius: '6px',
  },
}

const route = useRoute()
const router = useRouter()

const navItems = computed(() => [
  { path: '/dashboard', label: '仪表盘' },
  { path: '/mappings', label: '映射' },
  { path: '/cloudflare', label: 'Cloudflare' },
  { path: '/ddns', label: 'DDNS' },
  { path: '/settings', label: '设置' },
])

function isActive(path) {
  return route.path === path
}

function go(path) {
  if (route.path !== path) {
    router.push(path)
  }
}
</script>

<style>
* {
  box-sizing: border-box;
}

body {
  margin: 0;
  font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
}

#app {
  min-height: 100vh;
}

.app-layout {
  min-height: 100vh;
}

.app-header {
  height: 56px;
  padding: 0 20px;
  display: flex;
  align-items: center;
}

.app-header__inner {
  width: 100%;
  max-width: 1120px;
  margin: 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.app-header__left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.app-header__right {
  display: flex;
  align-items: center;
}

.app-header__logo-text {
  font-weight: 600;
  font-size: 1.1rem;
}

.app-content {
  padding: 20px;
}

.app-content__inner {
  width: 100%;
  max-width: 1120px;
  margin: 0 auto;
}
</style>
