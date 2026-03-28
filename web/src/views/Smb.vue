<template>
  <div>
    <page-header
      title="SMB 映射"
      description="管理 Windows SMB 共享映射（使用系统账户认证）。"
    >
      <template #actions>
        <n-space :size="8">
          <n-button type="primary" size="small" @click="openCreate">添加映射</n-button>
          <n-button size="small" :loading="syncingLocal" @click="syncLocalMappings">同步本地 SMB 映射</n-button>
          <n-button size="small" @click="loadMappings">刷新</n-button>
        </n-space>
      </template>
    </page-header>

    <n-alert type="info" class="page-section">
      SMB 使用 Windows 本机账户认证，不在 SkyLink 中维护独立 SMB 密码。
    </n-alert>

    <n-card class="page-section page-card">
      <n-spin :show="loading">
        <n-data-table :columns="columns" :data="list" :bordered="false" :single-line="false" />
      </n-spin>
    </n-card>

    <n-modal
      v-model:show="showEditor"
      preset="dialog"
      :title="editingId ? '编辑 SMB 映射' : '添加 SMB 映射'"
      positive-text="保存"
      @positive-click="onSave"
    >
      <n-form :model="form" label-placement="left" label-width="96" style="padding: 16px 0">
        <n-form-item label="名称" required>
          <n-input v-model:value="form.name" placeholder="例如：documents-smb" />
        </n-form-item>
        <n-form-item label="共享名" required>
          <n-input v-model:value="form.share_name" placeholder="例如：docs" />
        </n-form-item>
        <n-form-item label="本地目录" required>
          <local-path-input v-model="form.local_path" placeholder="例如：C:\Users\hey\Documents 或 C:/Users/hey/Documents" />
        </n-form-item>
        <n-form-item label="启用">
          <n-select v-model:value="form.enabled" :options="ENABLED_OPTIONS" />
        </n-form-item>
        <n-form-item label="写入模式">
          <n-select v-model:value="form.read_only" :options="READ_ONLY_OPTIONS" />
        </n-form-item>
        <n-form-item label="SMB 授权账户">
          <n-input
            v-model:value="form.grant_account"
            placeholder="留空：只读时使用 Everyone；读写使用系统默认。可填 Administrator 或 DOMAIN\\User"
          />
        </n-form-item>
      </n-form>
    </n-modal>
  </div>
</template>

<script setup>
import { computed, h, onMounted, ref } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NDataTable,
  NForm,
  NFormItem,
  NInput,
  NModal,
  NPopconfirm,
  NSelect,
  NSpace,
  NSpin,
  NTag,
} from 'naive-ui'
import api from '../api/client'
import PageHeader from '../components/PageHeader.vue'
import LocalPathInput from '../components/LocalPathInput.vue'
import { notifyError, notifySuccess } from '../ui/notify'
import { ENABLED_OPTIONS, READ_ONLY_OPTIONS } from '../constants/formOptions'
import { copyToClipboard } from '../utils/clipboard'

const loading = ref(false)
const list = ref([])
const showEditor = ref(false)
const editingId = ref(null)
const form = ref(newForm())
const syncingLocal = ref(false)
const requestOptions = { silentError: true }

const columns = computed(() => [
  { title: '名称', key: 'name', width: 140 },
  { title: '共享名', key: 'share_name', width: 120 },
  { title: '本地目录', key: 'local_path', ellipsis: true },
  {
    title: '授权账户',
    key: 'grant_account',
    width: 120,
    ellipsis: true,
    render: (row) => row.grant_account || '—',
  },
  {
    title: '状态',
    key: 'enabled',
    width: 100,
    render: (row) =>
      h(
        NTag,
        { type: row.enabled ? 'success' : 'default', size: 'small' },
        { default: () => (row.enabled ? '启用' : '停用') }
      ),
  },
  {
    title: '访问路径',
    key: 'unc',
    width: 300,
    render: (row) => {
      const unc = `\\\\${location.hostname}\\${row.share_name}`
      return h(NSpace, { size: 6, align: 'center', wrapItem: false }, {
        default: () => [
          h('span', {
            style: {
              display: 'inline-block',
              maxWidth: '250px',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            },
            title: unc,
          }, unc),
          h(
            NButton,
            { size: 'tiny', quaternary: true, title: '复制路径', onClick: () => copyText(unc) },
            { default: () => '📋' }
          ),
        ],
      })
    },
  },
  {
    title: '操作',
    key: 'actions',
    width: 320,
    render: (row) =>
      h(NSpace, { size: 4 }, {
        default: () => [
          h(NButton, { size: 'small', onClick: () => onToggleEnabled(row, !row.enabled) }, { default: () => (row.enabled ? '停用' : '启用') }),
          h(NButton, { size: 'small', onClick: () => onHealth(row) }, { default: () => '共享检查' }),
          h(NButton, { size: 'small', onClick: () => openEdit(row) }, { default: () => '编辑' }),
          h(
            NPopconfirm,
            { onPositiveClick: () => onDelete(row.id), positiveText: '删除', negativeText: '取消' },
            { trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => '删除' }), default: () => '确认删除？' }
          ),
        ],
      }),
  },
])

function newForm() {
  return {
    name: '',
    share_name: '',
    local_path: '',
    enabled: true,
    read_only: false,
    grant_account: '',
  }
}

async function loadMappings() {
  loading.value = true
  try {
    const { data } = await api.get('/smb/mappings', requestOptions)
    list.value = data?.list || []
  } catch (e) {
    notifyError('加载失败', e?.response?.data?.error || e?.message || '加载 SMB 列表失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  form.value = newForm()
  showEditor.value = true
}

function openEdit(row) {
  editingId.value = row.id
  form.value = {
    name: row.name || '',
    share_name: row.share_name || '',
    local_path: row.local_path || '',
    enabled: !!row.enabled,
    read_only: !!row.read_only,
    grant_account: row.grant_account || '',
  }
  showEditor.value = true
}

async function onSave() {
  if (!form.value.name.trim() || !form.value.share_name.trim() || !form.value.local_path.trim()) {
    notifyError('参数错误', '名称、共享名、本地目录是必填项')
    return false
  }
  try {
    if (editingId.value) {
      await api.put(`/smb/mappings/${editingId.value}`, form.value, requestOptions)
    } else {
      await api.post('/smb/mappings', form.value, requestOptions)
    }
    notifySuccess('已保存', 'SMB 映射配置已更新')
    await loadMappings()
    return true
  } catch (e) {
    notifyError('保存失败', e?.response?.data?.error || e?.message || '保存失败')
    return false
  }
}

async function onHealth(row) {
  try {
    const { data } = await api.get(`/smb/mappings/${row.id}/health`, requestOptions)
    if (data?.ok) notifySuccess('共享检查通过', data?.message || '')
    else notifyError('共享检查失败', data?.message || '共享不可用')
  } catch (e) {
    notifyError('共享检查失败', e?.response?.data?.error || e?.message || '请求失败')
  }
}

async function onDelete(id) {
  try {
    await api.delete(`/smb/mappings/${id}`, requestOptions)
    notifySuccess('已删除', 'SMB 映射已删除')
    await loadMappings()
  } catch (e) {
    notifyError('删除失败', e?.response?.data?.error || e?.message || '删除失败')
  }
}

async function onToggleEnabled(row, enabled) {
  try {
    const endpoint = enabled ? 'start' : 'stop'
    await api.post(`/smb/mappings/${row.id}/${endpoint}`, {}, requestOptions)
    notifySuccess(enabled ? '已启用' : '已停用', '')
    await loadMappings()
  } catch (e) {
    notifyError('操作失败', e?.response?.data?.error || e?.message || '操作失败')
  }
}

async function syncLocalMappings() {
  syncingLocal.value = true
  try {
    const { data } = await api.post('/smb/mappings/sync-local', {}, requestOptions)
    const added = Number(data?.added || 0)
    const updated = Number(data?.updated || 0)
    const skipped = Number(data?.skipped || 0)
    const errors = Array.isArray(data?.errors) ? data.errors : []
    if (errors.length > 0) {
      notifyError('同步完成（部分失败）', `新增 ${added}，更新 ${updated}，跳过 ${skipped}，失败 ${errors.length}`)
    } else {
      notifySuccess('同步完成', `新增 ${added}，更新 ${updated}，跳过 ${skipped}`)
    }
    await loadMappings()
  } catch (e) {
    notifyError('同步失败', e?.response?.data?.error || e?.message || '同步本地 SMB 映射失败')
  } finally {
    syncingLocal.value = false
  }
}

async function copyText(value) {
  try {
    await copyToClipboard(value)
    notifySuccess('已复制', value)
  } catch (_) {
    notifyError('复制失败', '请手动复制路径')
  }
}

onMounted(async () => {
  await loadMappings()
})
</script>
