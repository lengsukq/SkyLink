import { computed, onMounted, ref, watch, type Ref } from 'vue'
import { useRoute } from 'vue-router'
import api from '../api/client'
import { notifySuccess } from '../ui/notify'
import { ROUTE_PATHS } from '../constants/routes'
import type { CfAccount } from '../types/cfContext'

export function useCfAppContext() {
  const route = useRoute()

  const cfAccounts = ref<CfAccount[]>([])
  const cfCurrentAccountId = ref<number | null>(null)
  const cfAccountsLoading = ref(false)
  const isWindows = ref(false)

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

  async function refreshCfState(): Promise<void> {
    await Promise.all([fetchSettings(), fetchCfAccounts()])
  }

  function refreshWhenNotLogin(path: string) {
    if (path !== ROUTE_PATHS.login) {
      fetchSettings()
      fetchCfAccounts()
      fetchPlatformFlags()
    }
  }

  onMounted(() => {
    refreshWhenNotLogin(route.path)
  })

  watch(
    () => route.path,
    (path) => {
      refreshWhenNotLogin(path)
    },
  )

  const cfAccountOptions = computed(() =>
    cfAccounts.value.map((a) => ({
      label: a.name || `账号 #${a.id}`,
      value: a.id,
    })),
  )

  return {
    cfAccounts: cfAccounts as Ref<CfAccount[]>,
    cfCurrentAccountId,
    cfAccountsLoading,
    isWindows,
    cfAccountOptions,
    fetchSettings,
    fetchCfAccounts,
    onActivateCfAccount,
    refreshCfState,
  }
}
