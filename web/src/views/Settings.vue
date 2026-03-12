<template>
  <div>
    <page-header
      title="设置"
      description="管理控制台登录密码，以及一键映射时的默认 Cloudflare / FRP 配置。"
    />
    <n-card title="管理密码" style="max-width: 520px; margin-top: 16px">
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

    <n-card title="FRP / Cloudflare 默认配置" style="max-width: 520px; margin-top: 16px">
      <n-form :model="defaultsForm" label-placement="left" label-width="140">
        <n-form-item label="FRP 固定域名（CNAME 目标）">
          <n-input v-model:value="defaultsForm.frp_cname_target" placeholder="例如：xxx.sakurafrp.com（可留空）" />
        </n-form-item>
        <n-form-item label="CNAME 默认 proxied">
          <n-switch v-model:value="defaultsForm.cf_cname_proxied" />
        </n-form-item>
      </n-form>
      <n-space>
        <n-button type="primary" :loading="defaultsSaving" @click="saveDefaults">保存</n-button>
      </n-space>
      <p class="settings-hint">
        一键映射时如不填写 CNAME 目标，将自动使用这里的默认值。
      </p>
    </n-card>

    <n-card title="Cloudflare 账号管理" style="max-width: 720px; margin-top: 16px">
      <n-space vertical>
        <n-space align="center">
          <span>当前账号：</span>
          <n-select
            v-model:value="currentAccountId"
            :options="accountOptions"
            placeholder="请选择 Cloudflare 账号"
            style="width: 260px"
            :loading="accountsLoading"
            @update:value="onActivateAccount"
          />
          <n-button size="small" :loading="accountsLoading" @click="loadAccounts">刷新</n-button>
          <n-button size="small" type="primary" @click="openCreateAccount">新增账号</n-button>
        </n-space>
        <n-data-table :columns="accountColumns" :data="accounts" :bordered="false" />
      </n-space>
    </n-card>

    <n-modal
      v-model:show="showAccountModal"
      preset="dialog"
      :title="editingAccount ? '编辑 Cloudflare 账号' : '新增 Cloudflare 账号'"
      positive-text="保存"
      :positive-button-props="{ disabled: !zoneValidated }"
      :loading="accountSaving"
      @positive-click="onSaveAccount"
    >
      <n-form :model="accountForm" label-placement="left" label-width="120" style="padding: 16px 0">
        <n-form-item label="名称" required>
          <n-input v-model:value="accountForm.name" placeholder="例如：个人账号 / 公司账号" />
        </n-form-item>
        <n-form-item label="API Token" required>
          <n-input
            v-model:value="accountForm.api_token"
            type="password"
            show-password-on="click"
            placeholder="Cloudflare API Token（Zone.DNS Edit 权限）"
          />
        </n-form-item>
        <n-form-item label="默认 Zone">
          <n-select
            v-model:value="accountForm.zone_id"
            :options="zoneOptions"
            placeholder="请选择默认 Zone（需先点击检测连接加载列表）"
          />
          <template #feedback>
            <n-space align="center">
              <n-button size="tiny" tertiary :loading="zoneChecking" @click="validateZone">检测连接</n-button>
              <span v-if="zoneValidated">已通过校验</span>
            </n-space>
          </template>
        </n-form-item>
      </n-form>
    </n-modal>
  </div>
</template>

<script setup>
import { h, onMounted, reactive, ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NForm, NFormItem, NInput, NButton, NSpace, NSwitch, NSelect, NDataTable, NPopconfirm, NTag, NModal } from 'naive-ui'
import api from '../api/client'
import { notifySuccess } from '../ui/notify'
import PageHeader from '../components/PageHeader.vue'

const router = useRouter()
const loading = ref(false)
const pwForm = reactive({ old_password: '', new_password: '' })

const defaultsSaving = ref(false)
const defaultsForm = reactive({ frp_cname_target: '', cf_cname_proxied: true })

const accountsLoading = ref(false)
const accounts = ref([])
const currentAccountId = ref(null)
const showAccountModal = ref(false)
const accountSaving = ref(false)
const editingAccount = ref(null)
const accountForm = reactive({ name: '', api_token: '', zone_id: '' })
const zoneValidated = ref(false)
const zoneChecking = ref(false)
const availableZones = ref([])

const zoneOptions = computed(() =>
  (availableZones.value || []).map((z) => ({
    label: `${z.name} (${z.id})`,
    value: z.id,
  }))
)

const accountOptions = computed(() =>
  accounts.value.map((a) => ({
    label: a.name || `账号 #${a.id}`,
    value: a.id,
  }))
)

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
  accountForm.name = ''
  accountForm.api_token = ''
  accountForm.zone_id = ''
  showAccountModal.value = true
  zoneValidated.value = false
  availableZones.value = []
}

function openEditAccount(row) {
  editingAccount.value = row
  accountForm.name = row.name || ''
  accountForm.api_token = ''
  accountForm.zone_id = row.zone_id || ''
  showAccountModal.value = true
  zoneValidated.value = false
  availableZones.value = []
}

async function onSaveAccount() {
  if (!accountForm.name.trim() || (!editingAccount.value && !accountForm.api_token.trim())) {
    return false
  }
  if (!zoneValidated.value) {
    return false
  }
  accountSaving.value = true
  try {
    if (editingAccount.value) {
      await api.put(`/cf/accounts/${editingAccount.value.id}`, {
        name: accountForm.name.trim(),
        api_token: accountForm.api_token.trim() || undefined,
        zone_id: accountForm.zone_id.trim(),
      })
    } else {
      await api.post('/cf/accounts', {
        name: accountForm.name.trim(),
        api_token: accountForm.api_token.trim(),
        zone_id: accountForm.zone_id.trim(),
      })
    }
    await loadAccounts()
    showAccountModal.value = false
    notifySuccess('保存成功', 'Cloudflare 账号已更新')
    return true
  } finally {
    accountSaving.value = false
  }
}

async function validateZone() {
  if (!accountForm.api_token.trim()) return
  zoneChecking.value = true
  try {
    const { data } = await api.post('/cf/accounts/validate', {
      api_token: accountForm.api_token.trim(),
      zone_id: (accountForm.zone_id || '').trim(),
    })
    availableZones.value = data.zones || []
    if (data?.zone_id) {
      accountForm.zone_id = data.zone_id
    } else if (availableZones.value.length && !accountForm.zone_id) {
      // 未明确指定时，默认选第一个 Zone
      accountForm.zone_id = availableZones.value[0].id
    }
    zoneValidated.value = true
    notifySuccess('验证通过', 'API Token 与 Zone 设置已通过校验')
  } finally {
    zoneChecking.value = false
  }
}

async function deleteAccount(id) {
  await api.delete(`/cf/accounts/${id}`)
  if (currentAccountId.value === id) {
    currentAccountId.value = null
  }
  await loadAccounts()
}

async function onActivateAccount(id) {
  if (!id) return
  await api.put(`/cf/accounts/${id}/activate`)
  notifySuccess('已切换', '当前 Cloudflare 账号已更新')
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
