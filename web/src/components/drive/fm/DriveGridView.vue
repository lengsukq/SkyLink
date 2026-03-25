<template>
  <div class="grid">
    <div
      v-for="item in items"
      :key="item.path"
      class="cell"
      @dblclick="$emit('open', item)"
      @contextmenu.prevent="$emit('context', item, $event)"
      @click="$emit('select', item)"
    >
      <div class="thumb">
        <div v-if="thumbState(item.path) === 'loading'" class="thumb-loading">
          <n-spin size="small" />
        </div>
        <img
          v-if="thumbUrl(item.path)"
          :src="thumbUrl(item.path)!"
          loading="lazy"
          class="img"
          @load="onThumbLoaded(item.path)"
          @error="onThumbError(item.path)"
        />
      </div>
      <div class="name" :title="item.name">{{ item.name }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { NSpin } from 'naive-ui'
import { driveUserGetPreviewUrl } from '../../../api/driveUserClient'

type DriveEntry = {
  name: string
  path: string
  is_dir: boolean
  size_bytes: number
  modified_at: number
  ext: string
  type: string
}

const props = defineProps<{ items: DriveEntry[] }>()
defineEmits<{
  (e: 'open', item: DriveEntry): void
  (e: 'select', item: DriveEntry): void
  (e: 'context', item: DriveEntry, evt: MouseEvent): void
}>()

const state = reactive<Record<string, { url?: string; status: 'idle' | 'loading' | 'ok' | 'error' }>>({})

watch(
  () => props.items,
  (items) => {
    resetState()
    // prefetch a small window
    prefetch(items.slice(0, 24))
  },
  { immediate: true },
)

function resetState() {
  for (const k of Object.keys(state)) {
    delete state[k]
  }
}

function thumbUrl(path: string) {
  return state[path]?.url || ''
}

function thumbState(path: string) {
  return state[path]?.status || 'idle'
}

async function prefetch(list: DriveEntry[]) {
  for (const item of list) {
    if (item.is_dir) continue
    if (item.type !== 'image') continue
    if (state[item.path]?.status) continue
    state[item.path] = { status: 'loading' }
    try {
      const res = await driveUserGetPreviewUrl({ path: item.path })
      state[item.path] = { status: res?.url ? 'ok' : 'error', url: String(res?.url || '') }
    } catch {
      state[item.path] = { status: 'error' }
    }
  }
}

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
}
.thumb {
  position: relative;
  width: 100%;
  aspect-ratio: 1 / 1;
  border-radius: 8px;
  overflow: hidden;
  background: rgba(127, 127, 127, 0.08);
}
.thumb-loading {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}
.img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.name {
  margin-top: 6px;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>

