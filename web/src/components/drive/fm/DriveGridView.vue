<template>
  <div ref="gridRoot" class="grid">
    <div
      v-for="item in items"
      :key="item.path"
      class="cell"
      :class="{ selected: selectedPaths.includes(item.path) }"
      @dblclick="$emit('open', item)"
      @contextmenu.prevent="$emit('context', item, $event)"
      @click="$emit('select', item, $event)"
    >
      <div class="thumb">
        <template v-if="item.is_dir">
          <span class="icon-wrap" aria-hidden="true">
            <svg class="icon-svg icon-folder" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path
                d="M3 7a2 2 0 012-2h4l2 2h8a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V7z"
                stroke="currentColor"
                stroke-width="1.5"
                stroke-linejoin="round"
              />
            </svg>
          </span>
        </template>
        <template v-else-if="item.type === 'image'">
          <div class="thumb-hook" :data-path="item.path">
            <div v-if="thumbState(item.path) === 'loading'" class="thumb-loading">
              <n-spin size="small" />
            </div>
            <img
              v-if="thumbUrl(item.path)"
              :src="thumbUrl(item.path)!"
              loading="lazy"
              class="img"
              alt=""
              @load="onThumbLoaded(item.path)"
              @error="onThumbError(item.path)"
            />
          </div>
        </template>
        <template v-else>
          <span class="icon-wrap" aria-hidden="true" v-html="typeIconSvg(item)" />
        </template>
      </div>
      <div class="name" :title="item.name">{{ item.name }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { NSpin } from 'naive-ui'
import { driveUserGetPreviewUrlCached } from '../../../api/driveUserClient'
import type { DriveEntry } from '../../../types/drive'

const props = defineProps<{
  items: DriveEntry[]
  selectedPaths: string[]
}>()

defineEmits<{
  (e: 'open', item: DriveEntry): void
  (e: 'select', item: DriveEntry, evt: MouseEvent): void
  (e: 'context', item: DriveEntry, evt: MouseEvent): void
}>()

const gridRoot = ref<HTMLElement | null>(null)
const state = reactive<Record<string, { url?: string; status: 'idle' | 'loading' | 'ok' | 'error' }>>({})
let io: IntersectionObserver | null = null

function thumbUrl(path: string) {
  return state[path]?.url || ''
}

function thumbState(path: string) {
  return state[path]?.status || 'idle'
}

watch(
  () => props.items,
  (items) => {
    const paths = new Set(items.map((i) => i.path))
    for (const k of Object.keys(state)) {
      if (!paths.has(k)) delete state[k]
    }
    nextTick(setupObservers)
  },
  { deep: true },
)

function typeIconSvg(item: DriveEntry) {
  const t = item.type
  if (t === 'video') {
    return `<svg class="icon-svg icon-type" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M4 6a2 2 0 012-2h8a2 2 0 012 2v2.382l3-1.5V19l-3-1.5V19a2 2 0 01-2 2H6a2 2 0 01-2-2V6z" stroke="currentColor" stroke-width="1.5"/><path d="M10 10l4 2-4 2v-4z" fill="currentColor"/></svg>`
  }
  if (t === 'audio') {
    return `<svg class="icon-svg icon-type" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M9 18V5l12-2v13M9 18a3 3 0 103-3m9-1a3 3 0 103-3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`
  }
  if (t === 'document') {
    return `<svg class="icon-svg icon-type" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8l-6-6z" stroke="currentColor" stroke-width="1.5"/><path d="M14 2v6h6" stroke="currentColor" stroke-width="1.5"/></svg>`
  }
  if (t === 'archive') {
    return `<svg class="icon-svg icon-type" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M4 4h16v4H4V4zm0 8h16v8H4v-8zm4-4v4h8V8H8z" stroke="currentColor" stroke-width="1.5"/></svg>`
  }
  return `<svg class="icon-svg icon-type" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M4 4h16v16H4V4zm4 4h8v8H8V8z" stroke="currentColor" stroke-width="1.5" opacity=".5"/></svg>`
}

async function loadThumbForPath(path: string) {
  if (state[path]?.status && state[path].status !== 'idle') return
  state[path] = { status: 'loading' }
  try {
    const res = await driveUserGetPreviewUrlCached({ path })
    state[path] = { status: res?.url ? 'ok' : 'error', url: String(res?.url || '') }
  } catch {
    state[path] = { status: 'error' }
  }
}

function setupObservers() {
  io?.disconnect()
  if (!gridRoot.value) return
  io = new IntersectionObserver(
    (entries) => {
      for (const en of entries) {
        if (!en.isIntersecting) continue
        const path = (en.target as HTMLElement).dataset.path
        if (!path) continue
        const item = props.items.find((i) => i.path === path)
        if (!item || item.is_dir || item.type !== 'image') continue
        io?.unobserve(en.target)
        loadThumbForPath(path)
      }
    },
    { root: null, rootMargin: '120px', threshold: 0.01 },
  )
  gridRoot.value.querySelectorAll<HTMLElement>('.thumb-hook').forEach((el) => io!.observe(el))
}

onMounted(() => {
  nextTick(setupObservers)
})

onUnmounted(() => {
  io?.disconnect()
  io = null
})

function onThumbLoaded(path: string) {
  if (state[path]) state[path].status = 'ok'
}

function onThumbError(path: string) {
  if (state[path]) state[path].status = 'error'
}
</script>

<style scoped>
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 12px;
}
.cell {
  cursor: default;
  user-select: none;
  border-radius: 10px;
  padding: 6px;
  outline: 1px solid transparent;
  transition: outline-color 0.15s ease, background 0.15s ease;
}
.cell.selected {
  outline-color: rgba(59, 130, 246, 0.55);
  background: rgba(59, 130, 246, 0.06);
}
.thumb {
  position: relative;
  width: 100%;
  aspect-ratio: 1 / 1;
  border-radius: 8px;
  overflow: hidden;
  background: rgba(127, 127, 127, 0.08);
  display: flex;
  align-items: center;
  justify-content: center;
}
.thumb-hook {
  position: absolute;
  inset: 0;
}
.thumb-loading {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1;
}
.img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.icon-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56%;
  height: 56%;
  color: rgba(71, 85, 105, 0.85);
}
.icon-svg {
  width: 100%;
  height: 100%;
}
.icon-folder {
  color: rgba(245, 158, 11, 0.95);
}
.icon-type {
  color: rgba(100, 116, 139, 0.9);
}
.name {
  margin-top: 6px;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
:deep(.icon-type) {
  width: 100%;
  height: 100%;
}
</style>
