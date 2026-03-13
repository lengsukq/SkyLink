<template>
  <div>
    <n-space vertical>
      <n-h1>DDNS</n-h1>
      <n-space align="center">
        <n-button @click="load">刷新</n-button>
        <n-button @click="loadPublicIP">当前公网 IP</n-button>
        <n-space v-if="publicIPv4 || publicIPv6" vertical :size="4">
          <span v-if="publicIPv4">IPv4: {{ publicIPv4 }}</span>
          <span v-if="publicIPv6">IPv6: {{ publicIPv6 }}</span>
        </n-space>
      </n-space>
      <n-data-table :columns="columns" :data="list" :bordered="false" />
    </n-space>
  </div>
</template>

<script setup>
import { ref, h, onMounted } from 'vue'
import { NButton, NPopconfirm, NTag, NDataTable, NSpace, NH1 } from 'naive-ui'
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
