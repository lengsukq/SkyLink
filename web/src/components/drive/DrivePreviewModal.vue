<template>
  <n-modal v-model:show="show" preset="card" :title="title" style="width: min(980px, 96vw)">
    <div v-if="loading" class="loading">
      <n-spin size="large" />
    </div>

    <template v-else>
      <div v-if="!previewUrl" class="empty">无法生成预览链接</div>

      <div v-else class="preview">
        <div v-if="kind === 'image'" class="img-block">
          <div class="img-toolbar">
            <n-space align="center" :size="8">
              <n-button size="small" secondary :disabled="imgScale <= 0.25" @click="imgScale = Math.max(0.25, roundScale(imgScale - 0.25))">
                缩小
              </n-button>
              <n-button size="small" secondary :disabled="imgScale >= 3" @click="imgScale = Math.min(3, roundScale(imgScale + 0.25))">
                放大
              </n-button>
              <n-button size="small" quaternary @click="imgScale = 1">重置</n-button>
              <span class="hint">{{ Math.round(imgScale * 100) }}%</span>
            </n-space>
          </div>
          <div class="img-zoom-wrap">
            <div v-if="imgLoading" class="img-loading">
              <n-spin size="large" />
            </div>
            <img
              :src="previewUrl"
              class="img"
              :style="{ transform: `scale(${imgScale})` }"
              alt=""
              @load="imgLoading = false"
              @error="imgLoading = false"
            />
          </div>
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
      <n-space justify="space-between" style="width: 100%">
        <n-space v-if="previewNavTotal > 1" align="center" :size="8">
          <n-button size="small" secondary :disabled="previewNavIndex <= 0" @click="emit('nav-prev')">上一项</n-button>
          <span class="hint">{{ previewNavIndex + 1 }} / {{ previewNavTotal }}</span>
          <n-button size="small" secondary :disabled="previewNavIndex >= previewNavTotal - 1" @click="emit('nav-next')">
            下一项
          </n-button>
        </n-space>
        <span v-else />

        <n-space align="center" :size="8">
          <n-button secondary @click="onDownload" :disabled="!canDownload">下载</n-button>
          <n-button type="primary" @click="show = false">关闭</n-button>
        </n-space>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NButton, NModal, NSelect, NSpace, NSpin } from 'naive-ui'
import { DRIVE_VIDEO_RATE_OPTIONS } from '../../constants/drive'
import type { PreviewKind } from '../../types/drive'
import { useDrivePreviewUrl } from '../../composables/drive/useDrivePreviewUrl'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    path: string
    name: string
    kind: PreviewKind
    sizeBytes?: number
    previewNavTotal?: number
    previewNavIndex?: number
  }>(),
  {
    previewNavTotal: 0,
    previewNavIndex: 0,
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'nav-prev'): void
  (e: 'nav-next'): void
}>()

const show = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v),
})

const title = computed(() => `预览：${props.name}`)

const previewNavTotal = computed(() => props.previewNavTotal ?? 0)
const previewNavIndex = computed(() => props.previewNavIndex ?? 0)

const { loading, previewUrl, imgLoading, canDownload } = useDrivePreviewUrl({
  open: () => props.modelValue,
  path: () => props.path,
  kind: () => props.kind,
})

const imgScale = ref(1)

watch(
  () => props.path,
  () => {
    imgScale.value = 1
  },
)

watch(
  () => props.kind,
  () => {
    imgScale.value = 1
  },
)

function roundScale(n: number) {
  return Math.round(n * 100) / 100
}

const videoEl = ref<HTMLVideoElement | null>(null)
const playbackRate = ref<number>(1)
const rateOptions = DRIVE_VIDEO_RATE_OPTIONS

watch(
  () => playbackRate.value,
  (v) => {
    if (videoEl.value) videoEl.value.playbackRate = Number(v) || 1
  },
)

function onDownload() {
  const u = new URL('/api/drive/download', window.location.origin)
  u.searchParams.set('path', props.path || '')
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
.img-block {
  width: 100%;
}
.img-toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 8px;
}
.img-zoom-wrap {
  position: relative;
  max-height: 72vh;
  overflow: auto;
  display: flex;
  align-items: flex-start;
  justify-content: center;
}
.img-loading {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1;
  pointer-events: none;
}
.img {
  max-width: 100%;
  display: block;
  margin: 0 auto;
  transform-origin: center top;
  transition: transform 0.12s ease-out;
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
