import { onMounted, ref } from 'vue'
import api from '../../api/client'
import { notifyError, notifySuccess } from '../../ui/notify'
import { bytesToGbForInput, formatBytes, gbToBytes } from '../../utils/storage'
import { useDriveIssuedPassword } from './useDriveIssuedPassword'
import { getApiErrorMessage } from '../../utils/apiError'

export type DriveAccount = {
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

export function useDriveAccountsCrud() {
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

  async function refresh() {
    loading.value = true
    try {
      const res = await api.get('/drive/accounts')
      list.value = (res.data?.list || []) as DriveAccount[]
    } catch (e: unknown) {
      notifyError('加载失败', getApiErrorMessage(e))
    } finally {
      loading.value = false
    }
  }

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

  async function create() {
    const payload = {
      username: createForm.value.username.trim(),
      password: createForm.value.password,
      root_path: createForm.value.root_path.trim(),
      quota_bytes: gbToBytes(Number(createForm.value.quota_g) || 0),
      enabled: !!createForm.value.enabled,
    }
    if (!payload.username) return notifyError('创建失败', '用户名不能为空')
    if (!payload.password?.trim()) return notifyError('创建失败', '密码不能为空')
    if (!payload.root_path) return notifyError('创建失败', '根目录不能为空')

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
    const rootPath = editForm.value.root_path.trim()
    const quotaBytes = gbToBytes(Number(editForm.value.quota_g) || 0)
    if (!username) return notifyError('保存失败', '用户名不能为空')
    if (!rootPath) return notifyError('保存失败', '根目录不能为空')
    if (quotaBytes > 0 && quotaBytes < editUsedBytes.value) {
      return notifyError('保存失败', `配额须不少于已用量（${formatBytes(editUsedBytes.value)}）`)
    }

    savingEdit.value = true
    try {
      await api.put(`/drive/accounts/${editId.value}`, {
        username,
        root_path: rootPath,
        quota_bytes: quotaBytes,
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
      return notifyError('无效密码', '密码至少 6 位，或留空以随机生成')
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

  onMounted(() => refresh())

  return {
    loading,
    creating,
    savingEdit,
    resettingPwd,
    list,
    showCreate,
    createForm,
    showEdit,
    editId,
    editUsedBytes,
    editForm,
    showResetPwd,
    showIssued,
    issuedPassword,
    resetPwdInput,
    fillRandomPwd,
    copyIssuedPassword,
    openCreate,
    openEdit,
    resetEditForm,
    openResetPwd,
    refresh,
    create,
    saveEdit,
    submitResetPwd,
    recountUsed,
    remove,
  }
}

