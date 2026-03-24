<template>
  <div>
    <page-header
      title="Cloudflare 中心"
      description="统一管理 Cloudflare 账号、DNS 与 DDNS。"
    />

    <n-alert v-if="needAccountSetup" type="warning" class="page-section">
      请先在“账号与 DNS”中配置并激活 Cloudflare 账号，之后才能使用 DDNS。
    </n-alert>

    <n-card class="page-section page-card">
      <n-tabs v-model:value="activeTab" type="line" animated>
        <n-tab-pane name="cloudflare" tab="账号与 DNS">
          <cloudflare-view />
        </n-tab-pane>
        <n-tab-pane name="ddns" tab="DDNS" :disabled="needAccountSetup">
          <ddns-view />
        </n-tab-pane>
      </n-tabs>
    </n-card>
  </div>
</template>

<script setup>
import { computed, inject, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NAlert, NCard, NTabs, NTabPane } from 'naive-ui'
import PageHeader from '../components/PageHeader.vue'
import CloudflareView from './Cloudflare.vue'
import DDNSView from './DDNS.vue'

const route = useRoute()
const router = useRouter()
const cfCurrentAccountId = inject('cfCurrentAccountId', ref(null))
const cfAccounts = inject('cfAccounts', ref([]))

const TAB_CLOUDFLARE = 'cloudflare'
const TAB_DDNS = 'ddns'
const activeTab = ref('cloudflare')
const needAccountSetup = computed(
  () => !(cfAccounts.value || []).length || !cfCurrentAccountId.value
)

function normalizeTab(tab) {
  if (tab === TAB_DDNS) return TAB_DDNS
  return TAB_CLOUDFLARE
}

function getSafeTab(tab) {
  const normalized = normalizeTab(tab)
  if (normalized === TAB_DDNS && needAccountSetup.value) {
    return TAB_CLOUDFLARE
  }
  return normalized
}

watch(
  () => route.query.tab,
  (tab) => {
    const nextTab = getSafeTab(tab)
    if (activeTab.value !== nextTab) {
      activeTab.value = nextTab
    }
    if (tab !== nextTab) {
      router.replace({ query: { ...route.query, tab: nextTab } })
    }
  },
  { immediate: true }
)

watch(activeTab, (tab) => {
  const safeTab = getSafeTab(tab)
  if (safeTab !== tab) {
    activeTab.value = safeTab
    return
  }
  if (route.query.tab !== safeTab) {
    router.replace({ query: { ...route.query, tab: safeTab } })
  }
})

watch(needAccountSetup, (locked) => {
  if (locked && activeTab.value === TAB_DDNS) {
    activeTab.value = TAB_CLOUDFLARE
  }
})
</script>
