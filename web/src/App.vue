<template>
  <n-config-provider :theme-overrides="themeOverrides">
    <n-notification-provider>
      <notifier-bridge />
      <n-message-provider>
        <n-layout
          class="app-layout min-h-screen bg-transparent bg-gradient-to-b from-slate-100 via-sky-50/40 to-slate-100"
        >
          <n-layout-header
            v-if="!isLoginPage"
            class="app-header sticky top-0 z-40 border-b border-white/40 bg-white/65 shadow-sm shadow-slate-900/5 backdrop-blur-xl supports-[backdrop-filter]:bg-white/55"
            :bordered="false"
          >
            <div class="app-header__inner">
              <div class="app-header__left">
                <span class="app-header__logo-text tracking-tight">SkyLink</span>
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
          <n-layout-content
            :class="[
              'app-content px-4 pb-10 pt-6 sm:px-6',
              { 'app-content--login !p-0': isLoginPage },
            ]"
          >
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
import { computed, ref, provide } from 'vue'
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
import { ROUTE_PATHS } from './constants/routes'
import { cfAccountsKey, cfCurrentAccountIdKey, refreshCfStateKey } from './types/cfContext'
import { useCfAppContext } from './composables/useCfAppContext'

const themeOverrides = {
  common: {
    fontFamily: 'system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
    borderRadius: '16px',
    borderRadiusSmall: '12px',
    boxShadow1: '0 1px 2px 0 rgba(15, 23, 42, 0.05)',
    boxShadow2: '0 8px 24px -6px rgba(15, 23, 42, 0.08)',
    boxShadow3: '0 20px 48px -12px rgba(15, 23, 42, 0.12)',
  },
  Card: {
    borderRadius: '16px',
  },
  Button: {
    borderRadiusTiny: '10px',
    borderRadiusSmall: '12px',
    borderRadiusMedium: '14px',
    borderRadiusLarge: '16px',
  },
  Input: {
    borderRadius: '14px',
  },
  Select: {
    peers: {
      InternalSelection: {
        borderRadius: '14px',
      },
    },
  },
  DataTable: {
    borderRadius: '14px',
  },
  Dialog: {
    borderRadius: '18px',
  },
}

const route = useRoute()
const router = useRouter()

const {
  cfAccounts,
  cfCurrentAccountId,
  cfAccountsLoading,
  isWindows,
  cfAccountOptions,
  fetchSettings,
  fetchCfAccounts,
  onActivateCfAccount,
  refreshCfState,
} = useCfAppContext()

const showCfAccountModal = ref(false)

const navItems = computed(() => [
  { path: ROUTE_PATHS.dashboard, label: '仪表盘' },
  { path: ROUTE_PATHS.mappings, label: '映射' },
  { path: ROUTE_PATHS.cloudflareCenter, label: 'Cloudflare' },
  { path: ROUTE_PATHS.easyTier, label: 'EasyTier' },
  { path: ROUTE_PATHS.fileServices, label: '文件服务' },
  ...(isWindows.value ? [{ path: ROUTE_PATHS.windowsTools, label: 'Windows 工具' }] : []),
  { path: ROUTE_PATHS.settings, label: '设置' },
])

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

function onCfAccountSaved(newId?: number | null) {
  fetchCfAccounts().then(() => {
    if (newId) {
      onActivateCfAccount(newId)
    } else {
      fetchSettings()
    }
  })
}

provide(cfCurrentAccountIdKey, cfCurrentAccountId)
provide(cfAccountsKey, cfAccounts)
provide(refreshCfStateKey, refreshCfState)
</script>

<style>
* {
  box-sizing: border-box;
}

body {
  margin: 0;
}

#app {
  min-height: 100vh;
}

.app-layout {
  min-height: 100vh;
}

.app-header {
  height: 60px;
  padding: 0 20px;
  display: flex;
  align-items: center;
}

.app-header__inner {
  width: 100%;
  max-width: 1200px;
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
  font-weight: 650;
  font-size: 1.125rem;
  color: #0f172a;
}

.app-content__inner {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;
}

.app-content--login .app-content__inner {
  max-width: none;
  margin: 0;
}
</style>
