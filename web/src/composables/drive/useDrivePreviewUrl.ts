import { computed, ref, watch } from 'vue'
import { driveUserGetPreviewUrlCached } from '../../api/driveUserClient'

export function useDrivePreviewUrl(opts: { open: () => boolean; path: () => string; kind: () => string }) {
  const loading = ref(false)
  const previewUrl = ref('')
  const imgLoading = ref(false)

  watch(
    () => [opts.open(), opts.path(), opts.kind()] as const,
    async ([open]) => {
      if (!open) {
        previewUrl.value = ''
        return
      }
      await fetchPreviewUrl()
    },
  )

  async function fetchPreviewUrl() {
    loading.value = true
    try {
      const data = await driveUserGetPreviewUrlCached({ path: opts.path() })
      previewUrl.value = String(data?.url || '')
      imgLoading.value = opts.kind() === 'image' && !!previewUrl.value
    } finally {
      loading.value = false
    }
  }

  const canDownload = computed(() => !!opts.path())

  return { loading, previewUrl, imgLoading, canDownload }
}

