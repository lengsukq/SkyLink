<template>
  <div>
    <n-h1>仪表盘</n-h1>
    <n-alert v-if="loadError" type="warning" class="page-section">
      {{ loadError }}
    </n-alert>
    <n-grid cols="1 s:1 m:2 l:3 xl:4" x-gap="16" y-gap="16" class="page-section">
      <n-gi>
        <n-card title="映射数量" size="small" class="page-card">
          <n-statistic :value="stats.mappings_count" />
        </n-card>
      </n-gi>
      <n-gi>
        <n-card title="DDNS 配置数" size="small" class="page-card">
          <n-statistic :value="stats.ddns_count" />
        </n-card>
      </n-gi>
      <n-gi>
        <n-card title="Cloudflare 账号" size="small" class="page-card">
          <n-space vertical size="small">
            <n-statistic :value="stats.cf_accounts_count" />
            <n-space>
              <n-button type="primary" tag="a" :href="`#${ROUTE_PATHS.cloudflareCenter}`">管理 CF 账号</n-button>
              <n-button tag="a" :href="`#${ROUTE_PATHS.cloudflareCenter}`">查看 CF 记录</n-button>
            </n-space>
          </n-space>
        </n-card>
      </n-gi>
      <n-gi v-if="stats.is_windows">
        <storage-volumes-panel section-class="" />
      </n-gi>
      <n-gi>
        <n-card title="EasyTier 状态" size="small" class="page-card">
          <n-space vertical size="small">
            <n-space align="center">
              <span>启用状态：</span>
              <n-tag :type="stats.easytier_enabled ? 'success' : 'default'" size="small">
                {{ stats.easytier_enabled ? '已启用' : '未启用' }}
              </n-tag>
            </n-space>
            <n-space align="center">
              <span>Daemon：</span>
              <n-tag :type="stats.easytier_daemon_running ? 'success' : 'default'" size="small">
                {{ stats.easytier_daemon_running ? `运行中 (${stats.easytier_running_count || 0})` : '未运行' }}
              </n-tag>
            </n-space>
            <n-space align="center">
              <span>运行时：</span>
              <n-tag :type="stats.easytier_has_runtime ? 'success' : 'warning'" size="small">
                {{ stats.easytier_has_runtime ? '已安装' : '未准备' }}
              </n-tag>
            </n-space>
            <n-button size="small" type="primary" tag="a" :href="`#${ROUTE_PATHS.windowsTools}?tab=easytier`">
              Windows 工具 · EasyTier
            </n-button>
          </n-space>
        </n-card>
      </n-gi>
    </n-grid>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NCard, NGrid, NGi, NStatistic, NSpace, NButton, NH1, NTag, NAlert } from 'naive-ui'
import api from '../api/client'
import StorageVolumesPanel from '../components/StorageVolumesPanel.vue'
import { ROUTE_PATHS } from '../constants/routes'

type DashboardStats = {
  mappings_count: number
  ddns_count: number
  cf_accounts_count: number
  easytier_enabled: boolean
  easytier_daemon_running: boolean
  easytier_running_count: number
  easytier_has_runtime: boolean
  is_windows: boolean
}

const stats = ref<DashboardStats>({
  mappings_count: 0,
  ddns_count: 0,
  cf_accounts_count: 0,
  easytier_enabled: false,
  easytier_daemon_running: false,
  easytier_running_count: 0,
  easytier_has_runtime: false,
  is_windows: false,
})
const loadError = ref('')

onMounted(async () => {
  try {
    const { data } = await api.get('/stats', { silentError: true })
    stats.value = data as DashboardStats
    loadError.value = ''
  } catch (_) {
    loadError.value = '统计信息加载失败，请稍后重试。'
  }
})
</script>
