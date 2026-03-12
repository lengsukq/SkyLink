<template>
  <div>
    <n-h1>仪表盘</n-h1>
    <n-grid :cols="3" :x-gap="16" :y-gap="16" style="margin-top: 16px">
      <n-gi>
        <n-card title="映射数量" size="small">
          <n-statistic :value="stats.mappings_count" />
        </n-card>
      </n-gi>
      <n-gi>
        <n-card title="DDNS 配置数" size="small">
          <n-statistic :value="stats.ddns_count" />
        </n-card>
      </n-gi>
      <n-gi>
        <n-card title="快捷操作" size="small">
          <n-space>
            <n-button type="primary" tag="a" href="#/mappings">添加映射</n-button>
            <n-button tag="a" href="#/cloudflare">CF 记录</n-button>
          </n-space>
        </n-card>
      </n-gi>
    </n-grid>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { NCard, NGrid, NGi, NStatistic, NSpace, NButton, NH1 } from 'naive-ui'
import api from '../api/client'

const stats = ref({ mappings_count: 0, ddns_count: 0 })

onMounted(async () => {
  try {
    const { data } = await api.get('/stats')
    stats.value = data
  } catch (_) {
    // ignore
  }
})
</script>
