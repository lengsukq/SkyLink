<template>
  <div
    class="toolbar mb-4 rounded-2xl border border-white/45 bg-white/50 px-3 py-2.5 shadow-sm shadow-slate-900/5 backdrop-blur-md"
  >
    <n-space justify="space-between" align="center">
      <div class="title text-sm font-semibold tracking-tight text-slate-800">子账号</div>
      <n-space>
        <n-button secondary :loading="loading" @click="emit('refresh')">刷新</n-button>
        <n-button type="primary" @click="emit('create')">创建</n-button>
      </n-space>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { NSpace, NButton } from 'naive-ui'

defineProps<{
  loading: boolean
}>()

const emit = defineEmits<{
  refresh: []
  create: []
}>()
</script>
