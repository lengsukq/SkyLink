<template>
  <div>
    <drive-accounts-toolbar :loading="loading" @refresh="refresh" @create="openCreate" />

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
          <local-path-input v-model="createForm.root_path" placeholder="例如：D:\Drive\alice 或 /home/alice/drive" />
        </n-form-item>
        <n-form-item label="配额（G，0 表示不限制）">
          <n-input-number
            v-model:value="createForm.quota_g"
            :min="0"
            :step="1"
            :precision="2"
            placeholder="0"
            style="width: 100%"
          >
            <template #suffix>G</template>
          </n-input-number>
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

    <n-modal v-model:show="showEdit" preset="card" title="编辑子账号" style="max-width: 560px" @after-leave="resetEditForm">
      <n-alert v-if="editUsedBytes > 0" type="info" :bordered="false" style="margin-bottom: 12px">
        若设置配额 &gt; 0，须<strong>不少于当前已用量</strong>（{{ formatBytes(editUsedBytes) }}）。
      </n-alert>
      <n-form :model="editForm" label-placement="top">
        <n-form-item label="用户名">
          <n-input v-model:value="editForm.username" placeholder="用户名" />
        </n-form-item>
        <n-form-item label="根目录（本机路径）">
          <local-path-input v-model="editForm.root_path" placeholder="本机绝对路径" />
        </n-form-item>
        <n-form-item label="配额（G，0 表示不限制）">
          <n-input-number
            v-model:value="editForm.quota_g"
            :min="0"
            :step="1"
            :precision="2"
            placeholder="0"
            style="width: 100%"
          >
            <template #suffix>G</template>
          </n-input-number>
        </n-form-item>
        <n-form-item label="启用">
          <n-switch v-model:value="editForm.enabled" />
        </n-form-item>
      </n-form>

      <template #footer>
        <n-space justify="end">
          <n-button @click="showEdit = false">取消</n-button>
          <n-button type="primary" :loading="savingEdit" @click="saveEdit">保存</n-button>
        </n-space>
      </template>
    </n-modal>

    <n-modal v-model:show="showResetPwd" preset="card" title="重置登录密码" style="max-width: 480px">
      <n-alert type="warning" :bordered="false" style="margin-bottom: 12px">
        可输入新密码，或留空后点「确定」由服务器随机生成（仅随机时会再弹窗显示一次）。
      </n-alert>
      <n-form label-placement="top">
        <n-form-item label="新密码（至少 6 位）">
          <n-input
            v-model:value="resetPwdInput"
            type="password"
            show-password-on="click"
            placeholder="留空则随机生成"
          />
        </n-form-item>
        <n-form-item label=" ">
          <n-button secondary size="small" @click="fillRandomPwd">随机填入</n-button>
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showResetPwd = false">取消</n-button>
          <n-button type="primary" :loading="resettingPwd" @click="submitResetPwd">确定</n-button>
        </n-space>
      </template>
    </n-modal>

    <n-modal v-model:show="showIssued" preset="card" title="新密码（仅显示一次）" style="max-width: 680px">
      <n-alert type="warning" class="token-alert">请立即复制并妥善保存；服务器端只保存 hash，无法再次查看明文。</n-alert>
      <n-input :value="issuedPassword" readonly />
      <template #footer>
        <n-space justify="end">
          <n-button secondary @click="copyIssuedPassword">复制</n-button>
          <n-button type="primary" @click="showIssued = false">关闭</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import {
  NAlert,
  NButton,
  NDataTable,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NModal,
  NSpace,
  NSwitch,
} from 'naive-ui'
import LocalPathInput from '../LocalPathInput.vue'
import DriveAccountsToolbar from './DriveAccountsToolbar.vue'
import api from '../../api/client'
import { notifyError, notifySuccess } from '../../ui/notify'
import { formatBytes, formatBytesAsG, gbToBytes, bytesToGbForInput } from '../../utils/storage'
import { useDriveIssuedPassword } from '../../composables/drive/useDriveIssuedPassword'
import { getApiErrorMessage } from '../../utils/apiError'

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
const savingEdit = ref(false)
const resettingPwd = ref(false)
const list = ref<DriveAccount[]>([])

const showCreate = ref(false)
const createForm = ref({
  username: '',
  password: '',
  root_path: '',
  quota_g: 0 as number | null,
  enabled: true,
})

const showEdit = ref(false)
const editId = ref<number | null>(null)
const editUsedBytes = ref(0)
const editForm = ref({
  username: '',
  root_path: '',
  quota_g: 0 as number | null,
  enabled: true,
})

const showResetPwd = ref(false)
const resetAccountId = ref<number | null>(null)
const { resetPwdInput, showIssued, issuedPassword, fillRandomPwd, copyIssuedPassword } = useDriveIssuedPassword()

const columns = computed(() => [
  { title: '用户名', key: 'username', width: 140 },
  { title: '根目录', key: 'root_path', minWidth: 200 },
  {
    title: '用量 / 配额',
    key: 'quota',
    width: 190,
    render(row: DriveAccount) {
      const used = formatBytes(row.used_bytes)
      const quota = row.quota_bytes > 0 ? formatBytesAsG(row.quota_bytes) : '无限制'
      return `${used} / ${quota}`
    },
  },
  {
    title: '启用',
    key: 'enabled',
    width: 72,
    render(row: DriveAccount) {
      return row.enabled ? '是' : '否'
    },
  },
  {
    title: '操作',
    key: 'actions',
    width: 340,
    render(row: DriveAccount) {
      return h(
        NSpace,
        { size: 6, wrap: false },
        {
          default: () => [
            h(
              NButton,
              {
                size: 'small',
                secondary: true,
                onClick: () => openEdit(row),
              },
              { default: () => '编辑' },
            ),
            h(
              NButton,
              {
                size: 'small',
                secondary: true,
                onClick: () => openResetPwd(row.id),
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

function openCreate() {
  createForm.value = { username: '', password: '', root_path: '', quota_g: 0, enabled: true }
  showCreate.value = true
}

function openEdit(row: DriveAccount) {
  editId.value = row.id
  editUsedBytes.value = row.used_bytes
  editForm.value = {
    username: row.username,
    root_path: row.root_path,
    quota_g: bytesToGbForInput(row.quota_bytes),
    enabled: row.enabled,
  }
  showEdit.value = true
}

function resetEditForm() {
  editId.value = null
  editUsedBytes.value = 0
}

function openResetPwd(id: number) {
  resetAccountId.value = id
  resetPwdInput.value = ''
  showResetPwd.value = true
}

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
    quota_bytes: gbToBytes(Number(createForm.value.quota_g) || 0),
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
    await api.post('/drive/accounts', payload)
    notifySuccess('创建成功', '账号已创建')
    showCreate.value = false
    createForm.value = { username: '', password: '', root_path: '', quota_g: 0, enabled: true }
    await refresh()
  } catch (e: unknown) {
    notifyError('创建失败', getApiErrorMessage(e))
  } finally {
    creating.value = false
  }
}

async function saveEdit() {
  if (editId.value == null) return
  const username = editForm.value.username.trim()
  const root_path = editForm.value.root_path.trim()
  const quota_bytes = gbToBytes(Number(editForm.value.quota_g) || 0)
  if (!username) {
    notifyError('保存失败', '用户名不能为空')
    return
  }
  if (!root_path) {
    notifyError('保存失败', '根目录不能为空')
    return
  }
  if (quota_bytes > 0 && quota_bytes < editUsedBytes.value) {
    notifyError('保存失败', `配额须不少于已用量（${formatBytes(editUsedBytes.value)}）`)
    return
  }

  savingEdit.value = true
  try {
    await api.put(`/drive/accounts/${editId.value}`, {
      username,
      root_path,
      quota_bytes,
      enabled: !!editForm.value.enabled,
    })
    notifySuccess('已保存', '账号信息已更新')
    showEdit.value = false
    await refresh()
  } catch (e: unknown) {
    notifyError('保存失败', getApiErrorMessage(e))
  } finally {
    savingEdit.value = false
  }
}

async function submitResetPwd() {
  if (resetAccountId.value == null) return
  const raw = resetPwdInput.value.trim()
  if (raw.length > 0 && raw.length < 6) {
    notifyError('无效密码', '密码至少 6 位，或留空以随机生成')
    return
  }

  resettingPwd.value = true
  try {
    const body = raw.length > 0 ? { password: raw } : {}
    const { data } = await api.post(`/drive/accounts/${resetAccountId.value}/reset-password`, body)
    showResetPwd.value = false
    resetPwdInput.value = ''
    if (data?.password) {
      issuedPassword.value = String(data.password)
      showIssued.value = true
      notifySuccess('已重置', '请保存下方随机密码')
    } else {
      notifySuccess('已重置', '登录密码已更新')
    }
    await refresh()
  } catch (e: unknown) {
    notifyError('重置失败', getApiErrorMessage(e))
  } finally {
    resettingPwd.value = false
  }
}

async function recountUsed(id: number) {
  try {
    await api.post(`/drive/accounts/${id}/recount-used`)
    notifySuccess('已提交', '用量已重新计算')
    await refresh()
  } catch (e: unknown) {
    notifyError('操作失败', getApiErrorMessage(e))
  }
}

async function remove(id: number) {
  try {
    await api.delete(`/drive/accounts/${id}`)
    notifySuccess('删除成功', '子账号已删除')
    await refresh()
  } catch (e: unknown) {
    notifyError('删除失败', getApiErrorMessage(e))
  }
}

</script>

<style scoped>
.token-alert {
  margin-bottom: 12px;
}
</style>
