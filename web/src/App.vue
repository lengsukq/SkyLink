<template>
  <n-config-provider :theme-overrides="themeOverrides">
    <n-notification-provider>
      <notifier-bridge />
      <n-message-provider>
        <n-layout class="app-layout">
          <n-layout-header v-if="!isLoginPage" class="app-header" bordered>
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
                <n-space align="center" :size="8">
                  <n-select
                    v-model:value="cfCurrentAccountId"
                    :options="cfAccountOptions"
                    :placeholder="cfAccountOptions.length ? '未选择 CF 账号' : '请添加 CF 账号'"
                    style="width: 160px"
                    size="small"
                    :loading="cfAccountsLoading"
                    @update:value="onActivateCfAccount"
                  />
                  <n-button quaternary size="small" @click="onClickManageAccounts">
                    管理账号
                  </n-button>
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
                </n-space>
              </div>
            </div>
          </n-layout-header>
          <n-layout-content :class="['app-content', { 'app-content--login': isLoginPage }]">
            <div class="app-content__inner">
              <router-view />
            </div>
          </n-layout-content>
        </n-layout>
      </n-message-provider>
      <cf-account-form-modal
        v-model:show="showCfAccountModal"
        :editing-account="undefined"
        @saved="onCfAccountSaved"
      />
    </n-notification-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, watch, provide } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  NConfigProvider,
  NLayout,
  NLayoutHeader,
  NLayoutContent,
  NSpace,
  NButton,
  NSelect,
  NMessageProvider,
  NNotificationProvider,
} from 'naive-ui'
import NotifierBridge from './components/NotifierBridge.vue'
import CfAccountFormModal from './components/CfAccountFormModal.vue'
import api from './api/client'
import { notifySuccess } from './ui/notify'
import { ROUTE_PATHS } from './constants/routes'
import { cfAccountsKey, cfCurrentAccountIdKey, refreshCfStateKey, type CfAccount } from './types/cfContext'

const themeOverrides = {
  common: {
    borderRadius: '6px',
  },
}

const route = useRoute()
const router = useRouter()

const cfAccounts = ref<CfAccount[]>([])
const cfCurrentAccountId = ref<number | null>(null)
const cfAccountsLoading = ref(false)
const showCfAccountModal = ref(false)
const isWindows = ref(false)

const navItems = computed(() => [
  { path: ROUTE_PATHS.dashboard, label: '仪表盘' },
  { path: ROUTE_PATHS.mappings, label: '映射' },
  { path: ROUTE_PATHS.cloudflareCenter, label: 'Cloudflare' },
  { path: ROUTE_PATHS.easyTier, label: 'EasyTier' },
  { path: ROUTE_PATHS.fileServices, label: '文件服务' },
  ...(isWindows.value ? [{ path: ROUTE_PATHS.windowsTools, label: 'Windows 工具' }] : []),
  { path: ROUTE_PATHS.settings, label: '设置' },
])

const cfAccountOptions = computed(() =>
  cfAccounts.value.map((a) => ({
    label: a.name || `账号 #${a.id}`,
    value: a.id,
  }))
)

const isLoginPage = computed(
  () => route.path === ROUTE_PATHS.login || route.path === ROUTE_PATHS.driveLogin || route.path === ROUTE_PATHS.drivePortal,
)

function isActive(path: string) {
  return route.path === path
}

function go(path: string) {
  if (route.path !== path) {
    router.push(path)
  }
}

function onClickManageAccounts() {
  if (route.path !== ROUTE_PATHS.cloudflareCenter) {
    router.push({ path: ROUTE_PATHS.cloudflareCenter, query: { manage: '1' } })
  } else {
    router.push({ path: ROUTE_PATHS.cloudflareCenter, query: { manage: String(Date.now()) } })
  }
}

async function fetchSettings() {
  try {
    const { data } = await api.get('/settings')
    const id = data?.cf_current_account_id
    cfCurrentAccountId.value = id && Number(id) > 0 ? Number(id) : null
  } catch (_) {
    cfCurrentAccountId.value = null
  }
}

async function fetchCfAccounts() {
  cfAccountsLoading.value = true
  try {
    const { data } = await api.get('/cf/accounts')
    cfAccounts.value = data?.accounts || []
  } catch (_) {
    cfAccounts.value = []
  } finally {
    cfAccountsLoading.value = false
  }
}

async function fetchPlatformFlags() {
  try {
    const { data } = await api.get('/stats')
    isWindows.value = !!data?.is_windows
  } catch (_) {
    isWindows.value = false
  }
}

async function onActivateCfAccount(id: number | null) {
  if (!id) return
  try {
    await api.put(`/cf/accounts/${id}/activate`)
    cfCurrentAccountId.value = id
    notifySuccess('已切换', '当前 Cloudflare 账号已更新')
  } catch (_) {
    await fetchSettings()
  }
}

function onCfAccountSaved(newId?: number | null) {
  fetchCfAccounts().then(() => {
    if (newId) {
      onActivateCfAccount(newId)
    } else {
      fetchSettings()
    }
  })
}

async function refreshCfState(): Promise<void> {
  await Promise.all([fetchSettings(), fetchCfAccounts()])
}

provide(cfCurrentAccountIdKey, cfCurrentAccountId)
provide(cfAccountsKey, cfAccounts)
provide(refreshCfStateKey, refreshCfState)

onMounted(() => {
  if (route.path !== ROUTE_PATHS.login) {
    fetchSettings()
    fetchCfAccounts()
    fetchPlatformFlags()
  }
})

watch(
  () => route.path,
  (path) => {
    if (path !== ROUTE_PATHS.login) {
      fetchSettings()
      fetchCfAccounts()
      fetchPlatformFlags()
    }
  }
)
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

.app-content--login {
  padding: 0;
}

.app-content--login .app-content__inner {
  max-width: none;
  margin: 0;
}
</style>
