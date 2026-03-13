<template>
  <div>
    <page-header
      title="DDNS"
      description="查看并管理定时更新的 Cloudflare DDNS 任务，可快速查看当前公网 IP。"
    />
    <n-card class="page-section page-card">
      <n-space vertical :size="12">
        <n-space align="center" wrap :size="8">
          <n-button size="small" @click="load">刷新</n-button>
          <n-button size="small" @click="loadPublicIP">当前公网 IP</n-button>
          <n-space v-if="publicIPv4 || publicIPv6" vertical :size="2" class="ddns-ip-hint">
            <span v-if="publicIPv4">IPv4: {{ publicIPv4 }}</span>
            <span v-if="publicIPv6">IPv6: {{ publicIPv6 }}</span>
          </n-space>
        </n-space>
        <n-data-table :columns="columns" :data="list" :bordered="false" size="small" />
      </n-space>
    </n-card>
  </div>
</template>

<script setup>
import { ref, h, onMounted } from 'vue'
import { NButton, NPopconfirm, NTag, NDataTable, NSpace, NCard } from 'naive-ui'
import PageHeader from '../components/PageHeader.vue'
import api from '../api/client'

const list = ref([])
const publicIPv4 = ref('')
const publicIPv6 = ref('')

const columns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '类型', key: 'record_type', width: 70 },
  { title: 'Zone ID', key: 'zone_id', ellipsis: true },
  { title: '记录名', key: 'record_name' },
  { title: '间隔(分)', key: 'interval_min', width: 90 },
  { title: '启用', key: 'enabled', width: 70, render: (r) => h(NTag, { type: r.enabled ? 'success' : 'default' }, { default: () => (r.enabled ? '是' : '否') }) },
  { title: '上次 IP', key: 'last_ip' },
  { title: '操作', key: 'actions', width: 80, render: (row) => h(NPopconfirm, { onPositiveClick: () => remove(row.id) }, { trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => '删除' }), default: () => '确定删除？' }) },
]

async function load() {
  const { data } = await api.get('/ddns')
  list.value = data.list || []
}

async function loadPublicIP() {
  try {
    const { data } = await api.get('/ddns/ip')
    publicIPv4.value = data.ipv4 ?? data.ip ?? ''
    publicIPv6.value = data.ipv6 ?? ''
  } catch (_) {
    publicIPv4.value = ''
    publicIPv6.value = ''
  }
}

async function remove(id) {
  await api.delete(`/ddns/${id}`)
  await load()
}

onMounted(load)
</script>
