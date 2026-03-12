<template>
  <div>
    <n-space vertical>
      <n-h1>Cloudflare DNS</n-h1>
      <n-space>
        <n-select
          v-model:value="currentZoneId"
          :options="zoneOptions"
          placeholder="选择 Zone"
          style="width: 240px"
          @update:value="loadRecords"
        />
        <n-button :disabled="!currentZoneId" @click="loadRecords">刷新记录</n-button>
      </n-space>
      <n-data-table :columns="recordColumns" :data="records" :bordered="false" />
    </n-space>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { NSelect, NButton, NDataTable, NSpace, NH1 } from 'naive-ui'
import api from '../api/client'

const zones = ref([])
const records = ref([])
const currentZoneId = ref(null)

const zoneOptions = computed(() =>
  zones.value.map((z) => ({ label: z.name, value: z.id }))
)

const recordColumns = [
  { title: '类型', key: 'type', width: 80 },
  { title: '名称', key: 'name' },
  { title: '内容', key: 'content' },
  { title: 'Proxied', key: 'proxied', width: 80, render: (r) => (r.proxied ? '是' : '否') },
]

async function loadZones() {
  try {
    const { data } = await api.get('/cf/zones')
    zones.value = data.zones || []
    if (zones.value.length && !currentZoneId.value) currentZoneId.value = zones.value[0].id
  } catch (_) {}
}

async function loadRecords() {
  if (!currentZoneId.value) return
  try {
    const { data } = await api.get(`/cf/zones/${currentZoneId.value}/records`)
    records.value = data.records || []
  } catch (_) {
    records.value = []
  }
}

onMounted(() => {
  loadZones().then(() => loadRecords())
})
</script>
