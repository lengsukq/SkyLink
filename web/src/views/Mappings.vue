<template>
  <div>
    <n-space vertical>
      <n-h1>映射管理</n-h1>
      <n-space>
        <n-button type="primary" @click="showAdd = true">添加映射</n-button>
        <n-button @click="showOneClick = true">一键映射（含 CF CNAME）</n-button>
        <n-button @click="load">刷新</n-button>
      </n-space>
      <n-data-table :columns="columns" :data="list" :bordered="false" />
    </n-space>

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
      <n-form :model="oneClickForm" label-placement="left" label-width="100" style="padding: 16px 0">
        <n-form-item label="域名" required>
          <n-input v-model:value="oneClickForm.host" placeholder="xx.yourdomain.com" />
        </n-form-item>
        <n-form-item label="后端地址" required>
          <n-input v-model:value="oneClickForm.backend" placeholder="http://127.0.0.1:3000" />
        </n-form-item>
        <n-form-item label="Zone ID">
          <n-input v-model:value="oneClickForm.zone_id" placeholder="CF Zone ID" />
        </n-form-item>
        <n-form-item label="CNAME 目标">
          <n-input v-model:value="oneClickForm.cname_target" placeholder="樱花 frp 出口域名" />
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
import { ref, h, onMounted } from 'vue'
import { NButton, NPopconfirm, NSpace, NDataTable, NModal, NForm, NFormItem, NInput, NH1 } from 'naive-ui'
import api from '../api/client'

const list = ref([])
const showAdd = ref(false)
const showEdit = ref(false)
const showOneClick = ref(false)
const addForm = ref({ host: '', backend: '' })
const oneClickForm = ref({ host: '', backend: '', zone_id: '', cname_target: '' })
const editForm = ref({ id: null, host: '', backend: '' })

const columns = [
  { title: 'ID', key: 'id', width: 70 },
  { title: '域名', key: 'host' },
  { title: '后端', key: 'backend' },
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
  const { data } = await api.get('/mappings')
  list.value = data.list || []
}

async function onAdd() {
  if (!addForm.value.host?.trim() || !addForm.value.backend?.trim()) return false
  await api.post('/mappings', addForm.value)
  addForm.value = { host: '', backend: '' }
  await load()
  return true
}

async function onOneClick() {
  if (!oneClickForm.value.host?.trim() || !oneClickForm.value.backend?.trim()) return false
  await api.post('/mappings/one-click', oneClickForm.value)
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

onMounted(load)
</script>
