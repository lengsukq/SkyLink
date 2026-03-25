<template>
  <div>
    <div class="toolbar">
      <n-space justify="space-between" align="center">
        <div class="title">子账号</div>
        <n-space>
          <n-button secondary @click="refresh" :loading="loading">刷新</n-button>
          <n-button type="primary" @click="showCreate = true">创建</n-button>
        </n-space>
      </n-space>
    </div>

    <n-data-table
      :columns="columns"
      :data="list"
      :bordered="false"
      size="small"
      :single-line="false"
      :loading="loading"
    />

    <n-modal v-model:show="showCreate" preset="card" title="创建子账号" style="max-width: 560px">
      <n-form :model="createForm" label-placement="top">
        <n-form-item label="用户名">
          <n-input v-model:value="createForm.username" placeholder="例如：alice" />
        </n-form-item>
        <n-form-item label="密码">
          <n-input v-model:value="createForm.password" type="password" show-password-on="click" placeholder="设置初始密码" />
        </n-form-item>
        <n-form-item label="根目录（本机路径）">
          <n-input v-model:value="createForm.root_path" placeholder="例如：D:\\Drive\\alice" />
        </n-form-item>
        <n-form-item label="配额（字节，0 表示不限制）">
          <n-input-number v-model:value="createForm.quota_bytes" :min="0" style="width: 100%" />
        </n-form-item>
        <n-form-item label="启用">
          <n-switch v-model:value="createForm.enabled" />
        </n-form-item>
      </n-form>

      <template #footer>
        <n-space justify="end">
          <n-button @click="showCreate = false">取消</n-button>
          <n-button type="primary" :loading="creating" @click="create">创建</n-button>
        </n-space>
      </template>
    </n-modal>

    <n-modal v-model:show="showPassword" preset="card" title="新密码（仅显示一次）" style="max-width: 680px">
      <n-alert type="warning" class="token-alert">请立即复制并妥善保存；服务器端只保存 hash，无法再次查看明文。</n-alert>
      <n-input :value="issuedPassword" readonly />
      <template #footer>
        <n-space justify="end">
          <n-button secondary @click="copyIssuedPassword">复制</n-button>
          <n-button type="primary" @click="showPassword = false">关闭</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { NAlert, NButton, NDataTable, NForm, NFormItem, NInput, NInputNumber, NModal, NSpace, NSwitch } from 'naive-ui'
import api from '../../api/client'
import { notifyError, notifySuccess } from '../../ui/notify'
import { copyToClipboard } from '../../utils/clipboard'
import { formatBytes } from '../../utils/storage'

type DriveAccount = {
  id: number
  username: string
  root_path: string
  quota_bytes: number
  used_bytes: number
  enabled: boolean
  created_at: number
  updated_at: number
  last_used_at?: number | null
}

const loading = ref(false)
const creating = ref(false)
const list = ref<DriveAccount[]>([])

const showCreate = ref(false)
const createForm = ref({
  username: '',
  password: '',
  root_path: '',
  quota_bytes: 0,
  enabled: true,
})

const showPassword = ref(false)
const issuedPassword = ref('')

const columns = computed(() => [
  { title: '用户名', key: 'username', width: 160 },
  { title: '根目录', key: 'root_path', minWidth: 240 },
  {
    title: '用量 / 配额',
    key: 'quota',
    width: 200,
    render(row: DriveAccount) {
      const used = formatBytes(row.used_bytes)
      const quota = row.quota_bytes > 0 ? formatBytes(row.quota_bytes) : '无限制'
      return `${used} / ${quota}`
    },
  },
  {
    title: '启用',
    key: 'enabled',
    width: 90,
    render(row: DriveAccount) {
      return row.enabled ? '是' : '否'
    },
  },
  {
    title: '操作',
    key: 'actions',
    width: 260,
    render(row: DriveAccount) {
      return h(
        NSpace,
        { size: 8 },
        {
          default: () => [
            h(
              NButton,
              {
                size: 'small',
                secondary: true,
                onClick: () => resetPassword(row.id),
              },
              { default: () => '重置密码' },
            ),
            h(
              NButton,
              {
                size: 'small',
                secondary: true,
                onClick: () => recountUsed(row.id),
              },
              { default: () => '重算用量' },
            ),
            h(
              NButton,
              {
                size: 'small',
                tertiary: true,
                type: 'error',
                onClick: () => remove(row.id),
              },
              { default: () => '删除' },
            ),
          ],
        },
      )
    },
  },
])

onMounted(() => refresh())

async function refresh() {
  loading.value = true
  try {
    const res = await api.get('/drive/accounts')
    list.value = (res.data?.list || []) as DriveAccount[]
  } catch (e: any) {
    notifyError('加载失败', e?.message || String(e))
  } finally {
    loading.value = false
  }
}

async function create() {
  const payload = {
    username: createForm.value.username.trim(),
    password: createForm.value.password,
    root_path: createForm.value.root_path.trim(),
    quota_bytes: Number(createForm.value.quota_bytes) || 0,
    enabled: !!createForm.value.enabled,
  }
  if (!payload.username) {
    notifyError('创建失败', '用户名不能为空')
    return
  }
  if (!payload.password?.trim()) {
    notifyError('创建失败', '密码不能为空')
    return
  }
  if (!payload.root_path) {
    notifyError('创建失败', '根目录不能为空')
    return
  }

  creating.value = true
  try {
    const res = await api.post('/drive/accounts', payload)
    notifySuccess('创建成功', '账号已创建')
    showCreate.value = false
    createForm.value = { username: '', password: '', root_path: '', quota_bytes: 0, enabled: true }
    await refresh()
  } catch (e: any) {
    notifyError('创建失败', e?.response?.data?.error || e?.message || String(e))
  } finally {
    creating.value = false
  }
}

async function resetPassword(id: number) {
  try {
    const res = await api.post(`/drive/accounts/${id}/reset-password`)
    const password = String(res.data?.password || '')
    issuedPassword.value = password
    showPassword.value = true
    notifySuccess('重置成功', '已生成新密码（仅显示一次）')
  } catch (e: any) {
    notifyError('重置失败', e?.response?.data?.error || e?.message || String(e))
  }
}

async function recountUsed(id: number) {
  try {
    await api.post(`/drive/accounts/${id}/recount-used`)
    notifySuccess('已提交', '用量已重新计算')
    await refresh()
  } catch (e: any) {
    notifyError('操作失败', e?.response?.data?.error || e?.message || String(e))
  }
}

async function remove(id: number) {
  try {
    await api.delete(`/drive/accounts/${id}`)
    notifySuccess('删除成功', '子账号已删除')
    await refresh()
  } catch (e: any) {
    notifyError('删除失败', e?.response?.data?.error || e?.message || String(e))
  }
}

async function copyIssuedPassword() {
  try {
    await copyToClipboard(issuedPassword.value)
    notifySuccess('已复制', '密码已复制到剪贴板')
  } catch (e: any) {
    notifyError('复制失败', e?.message || String(e))
  }
}
</script>

<style scoped>
.toolbar {
  margin-bottom: 12px;
}
.title {
  font-size: 14px;
  font-weight: 600;
}
.token-alert {
  margin-bottom: 12px;
}
</style>

