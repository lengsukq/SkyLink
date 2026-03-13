<template>
  <div>
    <page-header
      title="设置"
      description="管理控制台登录密码，以及一键映射时的默认 Cloudflare / FRP 配置。"
    />
    <n-grid cols="1 s:1 m:2 l:2" x-gap="16" y-gap="16" class="page-section">
      <n-gi>
        <n-card title="管理密码" class="page-card">
          <n-form :model="pwForm" label-placement="left" label-width="100">
            <n-form-item label="当前密码" required>
              <n-input v-model:value="pwForm.old_password" type="password" show-password-on="click" placeholder="当前登录密码" />
            </n-form-item>
            <n-form-item label="新密码" required>
              <n-input v-model:value="pwForm.new_password" type="password" show-password-on="click" placeholder="设置一个新密码" />
            </n-form-item>
          </n-form>
          <n-space>
            <n-button type="primary" :loading="loading" @click="changePassword">修改密码</n-button>
            <n-button :disabled="loading || defaultsSaving" @click="logout">退出登录</n-button>
          </n-space>
          <p class="settings-hint">
            修改成功后会自动更新本地保存的登录密码，下次访问将直接使用新密码登录。
          </p>
        </n-card>
      </n-gi>

      <n-gi>
        <n-card title="FRP / Cloudflare 默认配置" class="page-card">
          <n-form :model="defaultsForm" label-placement="left" label-width="140">
            <n-form-item label="FRP 固定域名（CNAME 目标）">
              <n-input v-model:value="defaultsForm.frp_cname_target" placeholder="例如：xxx.sakurafrp.com" />
            </n-form-item>
            <n-form-item label="CNAME 默认 proxied">
              <n-switch v-model:value="defaultsForm.cf_cname_proxied" />
            </n-form-item>
          </n-form>
          <n-space>
            <n-button type="primary" :loading="defaultsSaving" @click="saveDefaults">保存</n-button>
          </n-space>
          <p class="settings-hint">
            一键映射功能会在 Cloudflare 中创建指向此 FRP 固定域名的 CNAME 记录；如未在此处配置，将无法使用一键映射创建 CNAME。
          </p>
        </n-card>
      </n-gi>
    </n-grid>

    <n-card title="Cloudflare 账号管理" class="page-section page-card">
      <n-space vertical>
        <p class="settings-hint" style="margin: 0 0 12px 0">
          当前使用的账号请在页面右上角切换。在此可新增、编辑或删除账号。
        </p>
        <n-space align="center">
          <n-button size="small" :loading="accountsLoading" @click="loadAccounts">刷新</n-button>
          <n-button size="small" type="primary" @click="openCreateAccount">新增账号</n-button>
        </n-space>
        <n-data-table :columns="accountColumns" :data="accounts" :bordered="false" />
      </n-space>
    </n-card>

    <cf-account-form-modal
      v-model:show="showAccountModal"
      :editing-account="editingAccount"
      @saved="onAccountSaved"
    />
  </div>
</template>

<script setup>
import { h, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NForm, NFormItem, NInput, NButton, NSpace, NSwitch, NDataTable, NPopconfirm, NGrid, NGi } from 'naive-ui'
import api from '../api/client'
import { notifySuccess } from '../ui/notify'
import PageHeader from '../components/PageHeader.vue'
import CfAccountFormModal from '../components/CfAccountFormModal.vue'

const router = useRouter()
const loading = ref(false)
const pwForm = reactive({ old_password: '', new_password: '' })

const defaultsSaving = ref(false)
const defaultsForm = reactive({ frp_cname_target: '', cf_cname_proxied: true })

const accountsLoading = ref(false)
const accounts = ref([])
const showAccountModal = ref(false)
const editingAccount = ref(null)

const accountColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '名称', key: 'name' },
  { title: '默认 Zone ID', key: 'zone_id' },
  {
    title: '操作',
    key: 'actions',
    width: 180,
    render: (row) =>
      h(NSpace, null, {
        default: () => [
          h(
            NButton,
            {
              size: 'small',
              onClick: () => openEditAccount(row),
            },
            { default: () => '编辑' }
          ),
          h(
            NPopconfirm,
            {
              positiveText: '删除',
              negativeText: '取消',
              onPositiveClick: () => deleteAccount(row.id),
            },
            {
              trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => '删除' }),
              default: () => '确定删除该账号？',
            }
          ),
        ],
      }),
  },
]

async function changePassword() {
  if (!pwForm.old_password.trim() || !pwForm.new_password.trim()) return
  loading.value = true
  try {
    await api.post('/auth/password', {
      old_password: pwForm.old_password.trim(),
      new_password: pwForm.new_password.trim(),
    })
    localStorage.setItem('skylink_token', pwForm.new_password.trim())
    pwForm.old_password = ''
    pwForm.new_password = ''
    notifySuccess('修改成功', '管理密码已更新')
  } finally {
    loading.value = false
  }
}

function logout() {
  localStorage.removeItem('skylink_token')
  router.replace('/login')
}

async function loadDefaults() {
  const { data } = await api.get('/settings')
  defaultsForm.frp_cname_target = (data?.frp_cname_target || '').toString()
  defaultsForm.cf_cname_proxied = !!data?.cf_cname_proxied
}

async function loadAccounts() {
  accountsLoading.value = true
  try {
    const { data } = await api.get('/cf/accounts')
    accounts.value = data.accounts || []
  } finally {
    accountsLoading.value = false
  }
}

function openCreateAccount() {
  editingAccount.value = null
  showAccountModal.value = true
}

function openEditAccount(row) {
  editingAccount.value = row
  showAccountModal.value = true
}

function onAccountSaved() {
  loadAccounts()
}

async function deleteAccount(id) {
  await api.delete(`/cf/accounts/${id}`)
  await loadAccounts()
}

async function saveDefaults() {
  defaultsSaving.value = true
  try {
    await api.put('/settings', {
      frp_cname_target: (defaultsForm.frp_cname_target || '').trim(),
      cf_cname_proxied: !!defaultsForm.cf_cname_proxied,
    })
    notifySuccess('保存成功', '默认配置已更新')
  } finally {
    defaultsSaving.value = false
  }
}

onMounted(async () => {
  await Promise.all([loadDefaults(), loadAccounts()])
})
</script>

<style>
.settings-hint {
  color: #666;
  font-size: 12px;
  margin-top: 8px;
  line-height: 1.6;
}
</style>
