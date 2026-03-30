import { computed, h, reactive, ref, type Ref } from 'vue'
import { NButton, type DataTableColumns, type SelectOption } from 'naive-ui'
import api from '../../api/client'
import { notifyError, notifySuccess } from '../../ui/notify'
import { fetchEasyTierPlatform, fetchEasyTierPlatforms, fetchEasyTierReleases } from './useEasyTierRuntimeCatalog'
import { installEasyTierRuntime, removeEasyTierRuntime, removeInstalledRuntime } from './useEasyTierRuntimeInstaller'
import type {
  EasyTierFormState,
  RuntimeInstalledItem,
  RuntimePlatformItem,
  RuntimeReleaseItem,
  EasyTierRuntimeInstalledResponse,
  EasyTierRuntimeListResponse,
} from './types'

type UseEasyTierRuntimeOptions = {
  form: Pick<EasyTierFormState, 'image_tag'>
  easytierHostSupported: Ref<boolean>
}

export function useEasyTierRuntime(options: UseEasyTierRuntimeOptions) {
  const platform = reactive({
    os: '',
    arch: '',
    label: '',
  })
  const releasesList = ref<RuntimeReleaseItem[]>([])
  const releasesError = ref('')
  const platformsList = ref<RuntimePlatformItem[]>([])
  const currentPlatformLabel = ref('')
  const releaseOptions = ref<SelectOption[]>([])
  const platformOptions = ref<SelectOption[]>([])
  const selectedVersion = ref<string | null>(null)
  const selectedPlatformKey = ref<string | null>(null)
  const runtimeInstalling = ref(false)
  const runtimeRemoving = ref(false)
  const runtimeInstalled = ref(false)
  const runtimeVersion = ref('')
  const platformError = ref('')
  const runtimeError = ref('')
  const installedList = ref<RuntimeInstalledItem[]>([])
  const removingInstalledKey = ref('')

  const installedOptionsForStatus = computed(() =>
    installedList.value.map((row) => ({
      label: `${row.version} (${row.os}/${row.arch})`,
      value: installedItemKey(row),
    }))
  )

  const selectedInstalledKey = computed({
    get() {
      const ver = selectedVersion.value || options.form.image_tag
      const plat = selectedPlatformKey.value
      if (!ver || !plat) return null
      const key = `${ver}-${plat.replace('/', '-')}`
      const found = installedList.value.some((row) => installedItemKey(row) === key)
      return found ? key : null
    },
    set(val) {
      if (!val) return
      const row = installedList.value.find((r) => installedItemKey(r) === val)
      if (row) {
        selectedVersion.value = row.version
        selectedPlatformKey.value = `${row.os}/${row.arch}`
      }
    },
  })

  const installedListColumns: DataTableColumns<RuntimeInstalledItem> = [
    { title: '版本', key: 'version', width: 100 },
    { title: '平台', key: 'platform', width: 110, render: (row: RuntimeInstalledItem) => `${row.os}/${row.arch}` },
    { title: '说明', key: 'current', width: 80, render: (row: RuntimeInstalledItem) => (isCurrentSelected(row) ? '当前使用' : '') },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (row: RuntimeInstalledItem) => {
        const key = installedItemKey(row)
        return h(
          NButton,
          {
            size: 'tiny',
            tertiary: true,
            loading: removingInstalledKey.value === key,
            disabled:
              !options.easytierHostSupported.value ||
              (removingInstalledKey.value !== '' && removingInstalledKey.value !== key),
            onClick: () => removeInstalledItem(row),
          },
          { default: () => '移除' }
        )
      },
    },
  ]

  function installedItemKey(row: RuntimeInstalledItem) {
    return `${row.version}-${row.os}-${row.arch}`
  }

  function isCurrentSelected(row: RuntimeInstalledItem) {
    const plat = selectedPlatformKey.value ? `${row.os}/${row.arch}` === selectedPlatformKey.value : false
    const ver = (selectedVersion.value || options.form.image_tag) && row.version === (selectedVersion.value || options.form.image_tag)
    return plat && ver
  }

  async function loadPlatform() {
    try {
      const data = await fetchEasyTierPlatform()
      platform.os = data.os
      platform.arch = data.arch
      platform.label = data.label
      options.easytierHostSupported.value = data.easytierHostSupported
      platformError.value = ''
    } catch (_) {
      platform.os = ''
      platform.arch = ''
      platform.label = ''
      options.easytierHostSupported.value = false
      platformError.value = '无法获取后端平台信息，运行时下载功能可能不可用。'
    }
  }

  async function loadReleases() {
    releasesError.value = ''
    try {
      const list = await fetchEasyTierReleases()
      releasesList.value = list
      releaseOptions.value = list.map((r) => ({ label: r.tag_name, value: r.tag_name }))
    } catch (_) {
      releasesList.value = []
      releaseOptions.value = []
      releasesError.value = '无法拉取 GitHub Releases 列表。'
    }
  }

  async function loadPlatforms() {
    try {
      const data = await fetchEasyTierPlatforms()
      platformsList.value = data.list
      currentPlatformLabel.value = data.currentLabel
      if (data.easytierHostSupported === true || data.easytierHostSupported === false) {
        options.easytierHostSupported.value = data.easytierHostSupported === true
      }
      platformOptions.value = data.list.map((p) => ({
        label: p.label || `${p.os}/${p.arch}`,
        value: `${p.os}/${p.arch}`,
      }))
    } catch (_) {
      platformsList.value = []
      platformOptions.value = []
    }
  }

  async function loadRuntimeInstalled() {
    const version = selectedVersion.value || options.form.image_tag
    if (!version || !selectedPlatformKey.value) {
      runtimeInstalled.value = false
      return
    }
    const [osVal, arch] = selectedPlatformKey.value.split('/')
    if (!osVal || !arch) {
      runtimeInstalled.value = false
      return
    }
    try {
      const { data } = await api.get<EasyTierRuntimeInstalledResponse>('/easytier/runtime/installed', {
        params: { version, os: osVal, arch },
      })
      runtimeInstalled.value = !!data?.installed
      if (data?.installed) {
        runtimeVersion.value = version
      }
    } catch (_) {
      runtimeInstalled.value = false
    }
  }

  async function loadInstalledList() {
    try {
      const { data } = await api.get<EasyTierRuntimeListResponse>('/easytier/runtime/list')
      installedList.value = data?.items || []
    } catch (_) {
      installedList.value = []
    }
  }

  async function removeInstalledItem(row: RuntimeInstalledItem) {
    const key = installedItemKey(row)
    removingInstalledKey.value = key
    try {
      await removeInstalledRuntime(row)
      notifySuccess('已移除', `${row.version} (${row.os}/${row.arch}) 已从本机删除。`)
      await loadInstalledList()
      await loadRuntimeInstalled()
    } catch (e) {
      const msg = e?.response?.data?.error || e?.message || '移除失败'
      notifyError('移除失败', msg)
    } finally {
      removingInstalledKey.value = ''
    }
  }

  function onVersionOrPlatformChange() {
    if (selectedVersion.value) {
      options.form.image_tag = selectedVersion.value
    }
    loadRuntimeInstalled()
  }

  async function installRuntime() {
    const version = selectedVersion.value || options.form.image_tag?.trim()
    if (!version) {
      notifyError('请选择版本', '请先从下拉框选择要下载的 EasyTier 版本。')
      return
    }
    if (!selectedPlatformKey.value) {
      notifyError('请选择平台', '请先从下拉框选择目标平台。')
      return
    }
    const [osVal, arch] = selectedPlatformKey.value.split('/')
    if (!osVal || !arch) {
      notifyError('平台格式错误', '请重新选择平台。')
      return
    }
    runtimeInstalling.value = true
    runtimeError.value = ''
    try {
      const data = await installEasyTierRuntime(
        version === 'latest' ? (releasesList.value[0]?.tag_name || 'latest') : version,
        osVal,
        arch
      )
      if (data?.installed) {
        runtimeVersion.value = data.version || version
        notifySuccess('运行时已准备就绪', `已为 ${selectedPlatformKey.value} 安装 EasyTier ${data.version || version}。`)
        runtimeError.value = ''
        await loadRuntimeInstalled()
        await loadInstalledList()
      }
    } catch (e) {
      const message =
        (e?.response?.data && (e.response.data.error || e.response.data.warning)) ||
        e?.message ||
        '下载 EasyTier 运行时失败。'
      runtimeError.value = `${message} 如当前官方未提供该平台的守护进程，可考虑手动安装并通过 SKYLINK_EASYTIER_DAEMON_PATH 指定路径。`
    } finally {
      runtimeInstalling.value = false
    }
  }

  async function removeRuntime() {
    const version = selectedVersion.value || options.form.image_tag?.trim()
    if (!version || !selectedPlatformKey.value) return
    const [osVal, arch] = selectedPlatformKey.value.split('/')
    if (!osVal || !arch) return
    runtimeRemoving.value = true
    runtimeError.value = ''
    try {
      await removeEasyTierRuntime(version, osVal, arch)
      notifySuccess('已移除', `已移除 ${selectedPlatformKey.value} 的 EasyTier ${version} 运行时。`)
      await loadRuntimeInstalled()
      await loadInstalledList()
    } catch (e) {
      const message = e?.response?.data?.error || e?.message || '移除失败。'
      runtimeError.value = message
    } finally {
      runtimeRemoving.value = false
    }
  }

  function ensureRuntimeDefaults() {
    if (!selectedVersion.value) {
      if (options.form.image_tag && releasesList.value.some((r) => r.tag_name === options.form.image_tag)) {
        selectedVersion.value = options.form.image_tag
      } else if (releasesList.value.length) {
        selectedVersion.value = releasesList.value[0].tag_name
        options.form.image_tag = releasesList.value[0].tag_name
      }
    }
    if (!selectedPlatformKey.value && platformOptions.value.length) {
      const cur = platform.os && platform.arch ? `${platform.os}/${platform.arch}` : ''
      const match = cur ? platformOptions.value.find((o: SelectOption) => o.value === cur) : undefined
      const picked = match?.value ?? platformOptions.value[0]?.value
      selectedPlatformKey.value = picked == null ? null : String(picked)
    }
  }

  return {
    platform,
    releasesList,
    releasesError,
    platformsList,
    currentPlatformLabel,
    releaseOptions,
    platformOptions,
    selectedVersion,
    selectedPlatformKey,
    runtimeInstalling,
    runtimeRemoving,
    runtimeInstalled,
    runtimeVersion,
    platformError,
    runtimeError,
    installedList,
    removingInstalledKey,
    installedOptionsForStatus,
    selectedInstalledKey,
    installedListColumns,
    loadPlatform,
    loadReleases,
    loadPlatforms,
    loadRuntimeInstalled,
    loadInstalledList,
    onVersionOrPlatformChange,
    installRuntime,
    removeRuntime,
    ensureRuntimeDefaults,
  }
}
