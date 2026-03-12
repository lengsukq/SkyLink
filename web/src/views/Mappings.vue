<template>
  <div>
    <page-header
      title="映射管理"
      description="管理域名到后端服务的映射，可选配合 Cloudflare CNAME 一键创建。"
    >
      <template #actions>
        <n-space>
          <n-button type="primary" @click="showAdd = true">添加映射</n-button>
          <n-button @click="showOneClick = true">一键映射（含 CF CNAME）</n-button>
          <n-button @click="load">刷新</n-button>
        </n-space>
      </template>
    </page-header>
    <n-spin :show="loading">
      <n-data-table
        :columns="columns"
        :data="list"
        :bordered="false"
        :single-line="false"
      >
        <template #empty>
          <empty-state
            title="当前暂无映射"
            description="可以为自己的域名创建到本地或内网服务的反向代理。"
            primary-text="添加映射"
            @primary="showAdd = true"
          />
        </template>
      </n-data-table>
    </n-spin>

    <n-modal v-model:show="showAdd" title="添加映射" preset="dialog" positive-text="添加" @positive-click="onAdd">
      <n-form :model="addForm" label-placement="left" label-width="80" style="padding: 16px 0">
        <n-form-item label="域名" path="host" required>
          <n-input v-model:value="addForm.host" placeholder="xx.yourdomain.com" />
        </n-form-item>
        <n-form-item label="后端" path="backend" required>
          <n-input v-model:value="addForm.backend" placeholder="http://127.0.0.1:3000" />
        </n-form-item>
      </n-form>
    </n-modal>

    <n-modal v-model:show="showOneClick" title="一键映射" preset="dialog" positive-text="添加并创建 CF CNAME" @positive-click="onOneClick">
      <n-form :model="oneClickForm" label-placement="left" label-width="120" style="padding: 16px 0">
        <n-form-item label="一级域名（Zone）">
          <n-select
            v-model:value="oneClickForm.zone_id"
            :options="zoneOptions"
            placeholder="选择 Cloudflare Zone（可选）"
            filterable
          />
        </n-form-item>
        <n-form-item label="二级域名">
          <n-input v-model:value="oneClickForm.host" placeholder="如 app（留空则使用根域名）" />
        </n-form-item>
        <n-form-item label="后端地址" required>
          <n-input v-model:value="oneClickForm.backend" placeholder="http://127.0.0.1:3000" />
        </n-form-item>
        <n-form-item label="CNAME 目标">
          <n-input v-model:value="oneClickForm.cname_target" placeholder="可留空，使用设置页默认 FRP 域名" />
        </n-form-item>
      </n-form>
    </n-modal>

    <n-modal v-model:show="showEdit" title="编辑映射" preset="dialog" positive-text="保存" @positive-click="onEdit">
      <n-form :model="editForm" label-placement="left" label-width="80" style="padding: 16px 0">
        <n-form-item label="域名">
          <n-input v-model:value="editForm.host" disabled />
        </n-form-item>
        <n-form-item label="后端" required>
          <n-input v-model:value="editForm.backend" placeholder="http://127.0.0.1:3000" />
        </n-form-item>
      </n-form>
    </n-modal>
  </div>
</template>

<script setup>
import { ref, h, onMounted, computed } from 'vue'
import { NButton, NPopconfirm, NSpace, NDataTable, NModal, NForm, NFormItem, NInput, NSelect, NSpin, NEllipsis } from 'naive-ui'
import api from '../api/client'
import PageHeader from '../components/PageHeader.vue'
import EmptyState from '../components/EmptyState.vue'

const list = ref([])
const loading = ref(false)
const showAdd = ref(false)
const showEdit = ref(false)
const showOneClick = ref(false)
const addForm = ref({ host: '', backend: '' })
const oneClickForm = ref({ host: '', backend: '', zone_id: '', cname_target: '' })
const editForm = ref({ id: null, host: '', backend: '' })

const zones = ref([])
const zoneOptions = computed(() => zones.value.map((z) => ({ label: z.name, value: z.id })))

const columns = [
  {
    title: 'ID',
    key: 'id',
    width: 70,
    className: 'text-secondary',
  },
  {
    title: '域名',
    key: 'host',
    ellipsis: true,
    render(row) {
      return h(
        NEllipsis,
        { style: 'max-width: 260px' },
        { default: () => row.host }
      )
    },
  },
  {
    title: '后端',
    key: 'backend',
    ellipsis: true,
    render(row) {
      return h(
        NEllipsis,
        { style: 'max-width: 320px' },
        { default: () => row.backend }
      )
    },
  },
  {
    title: '操作',
    key: 'actions',
    width: 160,
    render(row) {
      return h(NSpace, null, {
        default: () => [
          h(NButton, { size: 'small', onClick: () => { editForm.value = { id: row.id, host: row.host, backend: row.backend }; showEdit.value = true } }, { default: () => '编辑' }),
          h(NPopconfirm, {
            onPositiveClick: () => remove(row.id),
            positiveText: '删除',
            negativeText: '取消',
          }, { trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => '删除' }), default: () => '确定删除？' }),
        ],
      })
    },
  },
]

async function load() {
  loading.value = true
  try {
    const { data } = await api.get('/mappings')
    list.value = data.list || []
  } finally {
    loading.value = false
  }
}

async function loadZones() {
  try {
    const { data } = await api.get('/cf/zones')
    zones.value = data.zones || []
  } catch {
    zones.value = []
  }
}

async function onAdd() {
  if (!addForm.value.host?.trim() || !addForm.value.backend?.trim()) return false
  await api.post('/mappings', addForm.value)
  addForm.value = { host: '', backend: '' }
  await load()
  return true
}

async function onOneClick() {
  if (!oneClickForm.value.backend?.trim()) return false

  const zone = (zones.value || []).find((z) => z.id === oneClickForm.value.zone_id)
  const sub = (oneClickForm.value.host || '').trim()
  let fullHost = sub
  if (zone) {
    if (!sub) {
      fullHost = zone.name
    } else {
      fullHost = `${sub}.${zone.name}`
    }
  }

  await api.post('/mappings/one-click', {
    host: fullHost,
    backend: oneClickForm.value.backend,
    zone_id: oneClickForm.value.zone_id,
    cname_target: oneClickForm.value.cname_target,
  })
  await load()
  return true
}

async function onEdit() {
  if (!editForm.value.backend?.trim()) return false
  await api.put(`/mappings/${editForm.value.id}`, { backend: editForm.value.backend })
  await load()
  return true
}

async function remove(id) {
  await api.delete(`/mappings/${id}`)
  await load()
}

onMounted(() => {
  load()
  loadZones()
})
</script>
