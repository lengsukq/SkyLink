<template>
  <div>
    <page-header
      v-if="!embedded"
      title="WebDAV 映射"
      description="管理本地目录的 WebDAV 路由映射。访问入口：/api/webdav/{id}/..."
    >
      <template #actions>
        <n-space :size="8">
          <n-button type="primary" size="small" @click="openCreate">添加映射</n-button>
          <n-button size="small" @click="loadServices">刷新</n-button>
        </n-space>
      </template>
    </page-header>

    <n-space v-else :size="8" justify="end" class="page-section">
      <n-button type="primary" size="small" @click="openCreate">添加映射</n-button>
      <n-button size="small" @click="loadServices">刷新</n-button>
    </n-space>

    <n-alert type="warning" class="page-section">
      账号密码按你的要求为明文保存并可回显，请仅在受控环境使用。
    </n-alert>
    <n-alert type="info" class="page-section">
      访问链接用于 WebDAV 客户端（如资源管理器挂载、RaiDrive、Cyberduck）。浏览器直接打开可能出现 Method Not Allowed。
    </n-alert>

    <n-card class="page-section page-card">
      <n-spin :show="loading">
        <n-data-table :columns="columns" :data="list" :bordered="false" :single-line="false" />
      </n-spin>
    </n-card>

    <n-modal
      v-model:show="showEditor"
      preset="dialog"
      :title="editingId ? '编辑 WebDAV 映射' : '添加 WebDAV 映射'"
      positive-text="保存"
      @positive-click="onSave"
    >
      <n-form :model="form" label-placement="left" label-width="96" style="padding: 16px 0">
        <n-form-item label="名称" required>
          <n-input v-model:value="form.name" placeholder="例如：documents-dav" />
        </n-form-item>
        <n-form-item label="本地目录" required>
          <n-space vertical :size="6" style="width: 100%">
            <n-space :size="8">
              <n-input v-model:value="form.local_path" placeholder="例如：C:\\Users\\hey\\Documents" />
              <n-button size="small" @click="openDirectoryPicker">选择文件夹</n-button>
            </n-space>
            <span style="font-size: 12px; color: #666">
              浏览器模式通常无法直接读取绝对路径；若未自动填充完整路径，请手动输入。
            </span>
            <input
              ref="directoryInputRef"
              type="file"
              webkitdirectory
              directory
              multiple
              style="display: none"
              @change="onDirectoryPicked"
            />
          </n-space>
        </n-form-item>
        <n-form-item label="用户名" required>
          <n-input v-model:value="form.username" placeholder="WebDAV 用户名" />
        </n-form-item>
        <n-form-item label="密码" required>
          <n-input v-model:value="form.password" type="password" show-password-on="click" placeholder="WebDAV 密码" />
        </n-form-item>
        <n-form-item label="启用">
          <n-select v-model:value="form.enabled" :options="enabledOptions" />
        </n-form-item>
        <n-form-item label="写入模式">
          <n-select v-model:value="form.read_only" :options="readOnlyOptions" />
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
  NSelect,
  NSpace,
  NSpin,
  NTag,
  NPopconfirm,
} from 'naive-ui'
import api from '../api/client'
import PageHeader from '../components/PageHeader.vue'
import { notifyError, notifySuccess } from '../ui/notify'

defineProps({
  embedded: { type: Boolean, default: false },
})

const loading = ref(false)
const list = ref([])
const showEditor = ref(false)
const editingId = ref(null)
const form = ref(newForm())
const directoryInputRef = ref(null)

const enabledOptions = [
  { label: '启用', value: true },
  { label: '停用', value: false },
]
const readOnlyOptions = [
  { label: '读写', value: false },
  { label: '只读', value: true },
]

const webDavBaseOrigin = resolveWebDavBaseOrigin()

const columns = computed(() => [
  { title: '名称', key: 'name', width: 160 },
  { title: '本地目录', key: 'local_path', ellipsis: true },
  { title: '用户名', key: 'username', width: 120 },
  { title: '密码', key: 'password', width: 140 },
  {
    title: '状态',
    key: 'enabled',
    width: 120,
    render: (row) =>
      h(
        NTag,
        { type: row.enabled ? 'success' : 'default', size: 'small' },
        { default: () => (row.enabled ? '启用' : '停用') }
      ),
  },
  {
    title: '访问链接',
    key: 'link',
    width: 380,
    render: (row) => {
      const url = `${webDavBaseOrigin}/api/webdav/${row.id}/`
      return h(NSpace, { size: 6, align: 'center', wrapItem: false }, {
        default: () => [
          h('span', {
            style: {
              display: 'inline-block',
              maxWidth: '320px',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            },
            title: url,
          }, url),
          h(
            NButton,
            { size: 'tiny', quaternary: true, title: '复制链接', onClick: () => copyText(url) },
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
          h(NButton, { size: 'small', onClick: () => onHealth(row) }, { default: () => '路径检查' }),
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

function resolveWebDavBaseOrigin() {
  if (import.meta.env.DEV) {
    return `${window.location.protocol}//${window.location.hostname}:19080`
  }
  return window.location.origin
}

function newForm() {
  return {
    name: '',
    local_path: '',
    username: '',
    password: '',
    enabled: true,
    read_only: false,
  }
}

async function loadServices() {
  loading.value = true
  try {
    const { data } = await api.get('/webdav/mappings')
    list.value = data?.list || []
  } catch (e) {
    notifyError('加载失败', e?.response?.data?.error || e?.message || '加载服务列表失败')
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
    local_path: row.local_path || '',
    username: row.username || '',
    password: row.password || '',
    enabled: !!row.enabled,
    read_only: !!row.read_only,
  }
  showEditor.value = true
}

function openDirectoryPicker() {
  const input = directoryInputRef.value
  if (!input) {
    notifyError('当前环境不支持', '无法打开目录选择器，请手动填写本地目录路径。')
    return
  }
  input.value = ''
  input.click()
}

function onDirectoryPicked(event) {
  const input = event?.target
  const files = input?.files
  if (!files || files.length === 0) return

  const first = files[0]
  const relativePath = first.webkitRelativePath || ''
  const firstSegment = relativePath.split('/')[0] || ''

  // Electron/部分桌面容器会暴露 file.path，可推断绝对目录。
  const absoluteFilePath = typeof first.path === 'string' ? first.path : ''
  if (absoluteFilePath && relativePath) {
    const normalizedAbs = absoluteFilePath.replace(/\\/g, '/')
    const normalizedRel = relativePath.replace(/\\/g, '/')
    const suffix = `/${normalizedRel}`
    if (normalizedAbs.endsWith(suffix)) {
      const base = normalizedAbs.slice(0, normalizedAbs.length - suffix.length)
      form.value.local_path = base.replace(/\//g, '\\')
      notifySuccess('已填充目录', `已自动填充：${form.value.local_path}`)
      return
    }
  }

  if (firstSegment) {
    if (!form.value.local_path.trim()) {
      form.value.local_path = firstSegment
    }
    notifyError('无法获取绝对路径', '当前浏览器受限，已尝试提取目录名，请手动补全本地绝对路径。')
    return
  }

  notifyError('选择失败', '未能读取目录信息，请手动填写本地目录路径。')
}

async function onSave() {
  if (!form.value.name.trim() || !form.value.local_path.trim() || !form.value.username.trim() || !form.value.password.trim()) {
    notifyError('参数错误', '名称、本地目录、用户名、密码是必填项')
    return false
  }
  try {
    if (editingId.value) {
      await api.put(`/webdav/mappings/${editingId.value}`, form.value)
    } else {
      await api.post('/webdav/mappings', form.value)
    }
    notifySuccess('已保存', 'WebDAV 映射配置已更新')
    await loadServices()
    return true
  } catch (e) {
    notifyError('保存失败', e?.response?.data?.error || e?.message || '保存失败')
    return false
  }
}

async function onHealth(row) {
  try {
    const { data } = await api.get(`/webdav/mappings/${row.id}/health`)
    if (data?.ok) notifySuccess('路径检查通过', data?.message || '')
    else notifyError('路径检查失败', data?.message || '目录不可用')
  } catch (e) {
    notifyError('路径检查失败', e?.response?.data?.error || e?.message || '请求失败')
  }
}

async function onDelete(id) {
  try {
    await api.delete(`/webdav/mappings/${id}`)
    notifySuccess('已删除', '服务配置已删除')
    await loadServices()
  } catch (e) {
    notifyError('删除失败', e?.response?.data?.error || e?.message || '删除失败')
  }
}

async function onToggleEnabled(row, enabled) {
  try {
    const endpoint = enabled ? 'start' : 'stop'
    await api.post(`/webdav/mappings/${row.id}/${endpoint}`)
    notifySuccess(enabled ? '已启用' : '已停用', '')
    await loadServices()
  } catch (e) {
    notifyError('操作失败', e?.response?.data?.error || e?.message || '操作失败')
  }
}

async function copyText(value) {
  try {
    await navigator.clipboard.writeText(value)
    notifySuccess('已复制', value)
  } catch (_) {
    notifyError('复制失败', '请手动复制链接')
  }
}

onMounted(async () => {
  await loadServices()
})
</script>
