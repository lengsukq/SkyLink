<template>
  <n-card size="small" class="sidebar" :bordered="false">
    <template #header>
      <div class="title">详情</div>
    </template>

    <div v-if="!item" class="empty">未选择文件</div>

    <template v-else>
      <div class="name">{{ item.name }}</div>
      <div class="kv">
        <div class="k">类型</div>
        <div class="v">{{ item.is_dir ? '目录' : item.type }}</div>
      </div>
      <div class="kv">
        <div class="k">大小</div>
        <div class="v">{{ item.is_dir ? '-' : formatBytes(item.size_bytes) }}</div>
      </div>
      <div class="kv">
        <div class="k">修改时间</div>
        <div class="v">{{ formatTime(item.modified_at) }}</div>
      </div>
      <div class="kv">
        <div class="k">路径</div>
        <div class="v mono">{{ item.path }}</div>
      </div>

      <n-divider />

      <n-space vertical>
        <n-button secondary :disabled="item.is_dir" @click="$emit('preview')">预览</n-button>
        <n-button secondary :disabled="item.is_dir" @click="$emit('download')">下载</n-button>
        <n-button secondary @click="$emit('rename')">重命名</n-button>
        <n-button secondary @click="$emit('cut')">剪切</n-button>
        <n-button secondary @click="$emit('copy-path')">复制路径</n-button>
        <n-button tertiary type="error" @click="$emit('delete')">删除</n-button>
      </n-space>
    </template>
  </n-card>
</template>

<script setup lang="ts">
import { NButton, NCard, NDivider, NSpace } from 'naive-ui'
import { formatBytes } from '../../../utils/storage'

type DriveEntry = {
  name: string
  path: string
  is_dir: boolean
  size_bytes: number
  modified_at: number
  ext: string
  type: string
}

defineProps<{ item: DriveEntry | null }>()
defineEmits<{
  (e: 'preview'): void
  (e: 'download'): void
  (e: 'delete'): void
  (e: 'rename'): void
  (e: 'cut'): void
  (e: 'copy-path'): void
}>()

function formatTime(ts: number) {
  if (!ts) return '-'
  const d = new Date(ts * 1000)
  return d.toLocaleString()
}
</script>

<style scoped>
.sidebar {
  position: sticky;
  top: 12px;
}
.title {
  font-weight: 600;
}
.empty {
  opacity: 0.75;
  padding: 8px 0;
}
.name {
  font-weight: 700;
  margin-bottom: 8px;
  word-break: break-all;
}
.kv {
  display: grid;
  grid-template-columns: 72px 1fr;
  gap: 8px;
  margin: 6px 0;
}
.k {
  opacity: 0.75;
}
.v {
  word-break: break-all;
}
.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
}
</style>

