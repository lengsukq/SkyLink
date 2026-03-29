<template>
  <n-layout
    class="app-layout min-h-screen bg-transparent bg-gradient-to-b from-slate-100 via-sky-50 via-sky-100/50 to-slate-100"
  >
    <n-layout-header
      class="app-header sticky top-0 z-40 border-b border-white/40 bg-white/65 shadow-sm shadow-slate-900/5 backdrop-blur-xl supports-[backdrop-filter]:bg-white/55"
      :bordered="false"
    >
      <div class="app-header__inner">
        <div class="app-header__left">
          <sky-link-logo size="sm" :to="ROUTE_PATHS.dashboard" />
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
    <n-layout-content class="app-content px-4 pb-10 pt-6 sm:px-6">
      <div class="app-content__inner">
        <router-view />
      </div>
    </n-layout-content>
    <cf-account-form-modal
      v-model:show="showCfAccountModal"
      :editing-account="undefined"
      @saved="onCfAccountSaved"
    />
  </n-layout>
</template>

<script setup lang="ts">
import { computed, provide, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NLayout, NLayoutHeader, NLayoutContent, NSpace, NButton, NSelect } from 'naive-ui'
import CfAccountFormModal from '../components/CfAccountFormModal.vue'
import SkyLinkLogo from '../components/SkyLinkLogo.vue'
import { ROUTE_PATHS } from '../constants/routes'
import { useCfAppContext } from '../composables/useCfAppContext'
import { getAdminNavItems } from '../router/nav'
import { cfAccountsKey, cfCurrentAccountIdKey, refreshCfStateKey } from '../types/cfContext'

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

provide(cfCurrentAccountIdKey, cfCurrentAccountId)
provide(cfAccountsKey, cfAccounts)
provide(refreshCfStateKey, refreshCfState)

const showCfAccountModal = ref(false)

const navItems = computed(() => getAdminNavItems(router, isWindows.value))

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
</script>

<style scoped>
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

.app-content__inner {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;
}
</style>
