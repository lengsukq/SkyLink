<template>
  <n-modal v-model:show="show" preset="card" :title="title" style="width: min(980px, 96vw)">
    <div v-if="loading" class="loading">
      <n-spin size="large" />
    </div>

    <template v-else>
      <div v-if="!previewUrl" class="empty">无法生成预览链接</div>

      <div v-else class="preview">
        <div v-if="kind === 'image'" class="img-wrap">
          <div v-if="imgLoading" class="img-loading">
            <n-spin size="large" />
          </div>
          <img :src="previewUrl" class="img" @load="imgLoading = false" @error="imgLoading = false" />
        </div>

        <div v-else-if="kind === 'video'" class="video-wrap">
          <div class="video-toolbar">
            <n-space align="center" :size="8">
              <span class="hint">倍速</span>
              <n-select v-model:value="playbackRate" :options="rateOptions" size="small" style="width: 120px" />
            </n-space>
          </div>
          <video ref="videoEl" :src="previewUrl" class="video" controls playsinline />
        </div>

        <audio v-else-if="kind === 'audio'" :src="previewUrl" controls class="audio" />

        <iframe v-else-if="kind === 'pdf'" :src="previewUrl" class="pdf" />

        <div v-else class="empty">不支持预览该类型</div>
      </div>
    </template>

    <template #footer>
      <n-space justify="end">
        <n-button secondary @click="onDownload" :disabled="!canDownload">下载</n-button>
        <n-button type="primary" @click="show = false">关闭</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NButton, NModal, NSelect, NSpace, NSpin } from 'naive-ui'
import api from '../../api/client'

type PreviewKind = 'image' | 'video' | 'audio' | 'pdf' | 'unknown'

const props = defineProps<{
  modelValue: boolean
  path: string
  name: string
  kind: PreviewKind
  sizeBytes?: number
}>()

const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const show = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v),
})

const title = computed(() => `预览：${props.name}`)

const loading = ref(false)
const previewUrl = ref('')
const imgLoading = ref(false)

const videoEl = ref<HTMLVideoElement | null>(null)
const playbackRate = ref<number>(1)
const rateOptions = [
  { label: '0.5x', value: 0.5 },
  { label: '1x', value: 1 },
  { label: '1.25x', value: 1.25 },
  { label: '1.5x', value: 1.5 },
  { label: '2x', value: 2 },
]

const canDownload = computed(() => !!props.path && !!props.name)

watch(
  () => [props.modelValue, props.path, props.kind] as const,
  async ([open]) => {
    if (!open) {
      previewUrl.value = ''
      return
    }
    await fetchPreviewUrl()
  },
)

watch(
  () => playbackRate.value,
  (v) => {
    if (videoEl.value) videoEl.value.playbackRate = Number(v) || 1
  },
)

async function fetchPreviewUrl() {
  loading.value = true
  try {
    const expires = computeExpires(props.kind, props.sizeBytes || 0)
    const { data } = await api.get('/drive/preview-url', {
      params: { path: props.path, expires },
      silentError: true,
    } as any)
    previewUrl.value = String(data?.url || '')
    imgLoading.value = props.kind === 'image' && !!previewUrl.value
  } finally {
    loading.value = false
  }
}

function computeExpires(kind: PreviewKind, size: number) {
  // Backend defaults by file size, but frontend can hint a longer value.
  if (kind === 'video' && size >= 1024 * 1024 * 1024) return 3 * 60 * 60
  if (kind === 'video' && size >= 200 * 1024 * 1024) return 2 * 60 * 60
  return 60 * 60
}

function onDownload() {
  const u = new URL('/api/drive/download', window.location.origin)
  u.searchParams.set('path', props.path || '')
  // download endpoint requires Authorization header; we fallback to opening preview url in new tab if present.
  if (previewUrl.value) {
    window.open(previewUrl.value, '_blank', 'noopener,noreferrer')
  } else {
    window.open(u.toString(), '_blank', 'noopener,noreferrer')
  }
}
</script>

<style scoped>
.loading {
  min-height: 260px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.preview {
  min-height: 260px;
}
.hint {
  opacity: 0.8;
  font-size: 12px;
}
.img-wrap {
  position: relative;
}
.img-loading {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}
.img {
  max-width: 100%;
  max-height: 72vh;
  display: block;
  margin: 0 auto;
}
.video-wrap {
  width: 100%;
}
.video-toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 8px;
}
.video {
  width: 100%;
  max-height: 72vh;
}
.audio {
  width: 100%;
}
.pdf {
  width: 100%;
  height: 72vh;
  border: 0;
}
.empty {
  padding: 16px;
  opacity: 0.8;
}
</style>

