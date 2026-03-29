<template>
  <n-card
    v-if="visible"
    title="本地磁盘"
    size="small"
    class="page-card"
    :class="sectionClass"
  >
    <n-space vertical :size="12">
      <div v-for="v in volumes" :key="v.path" class="storage-vol">
        <div class="storage-vol__head">
          <span class="storage-vol__path">{{ v.path }}</span>
          <span v-if="v.label" class="storage-vol__label">{{ v.label }}</span>
        </div>
        <n-progress type="line" :percentage="percentUsed(v)" indicator-placement="inside" />
        <div class="storage-vol__meta">
          已用 {{ formatBytes(v.used_bytes) }} / 共 {{ formatBytes(v.total_bytes) }}；可用
          {{ formatBytes(v.free_bytes) }}
        </div>
      </div>
    </n-space>
  </n-card>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { NCard, NProgress, NSpace } from 'naive-ui'
import api from '../api/client'
import { formatBytes, percentUsed } from '../utils/storage'

defineProps({
  sectionClass: {
    type: String,
    default: 'page-section',
  },
})

const supported = ref(false)
const volumes = ref([])
const loaded = ref(false)

const visible = computed(
  () => loaded.value && supported.value && Array.isArray(volumes.value) && volumes.value.length > 0
)

onMounted(async () => {
  try {
    const { data } = await api.get('/system/volumes')
    supported.value = !!data?.supported
    volumes.value = data?.volumes || []
  } catch (_) {
    supported.value = false
    volumes.value = []
  } finally {
    loaded.value = true
  }
})
</script>

<style scoped>
.storage-vol__head {
  display: flex;
  align-items: baseline;
  gap: 10px;
  flex-wrap: wrap;
}
.storage-vol__path {
  font-weight: 600;
}
.storage-vol__label {
  font-size: 12px;
  color: #64748b;
}
.storage-vol__meta {
  font-size: 12px;
  color: #64748b;
}
</style>
