<template>
  <div>
    <template v-if="needCfAccountSetup">
      <page-header
        title="Cloudflare DNS"
        description="在选定的 Zone 下管理 DNS 记录，可搜索和分页浏览。"
      />
      <n-card v-if="!hasAnyCfAccount" class="page-section page-card">
        <empty-state
          title="需要添加 Cloudflare 账号"
          description="请先添加 Cloudflare 账号并选择 Zone，才能在此管理 DNS 记录。"
          primary-text="添加 Cloudflare 账号"
          @primary="openAccountsManager"
        />
      </n-card>
      <n-card v-else class="page-section page-card">
        <empty-state
          title="请选择当前账号"
          description="您已添加了 Cloudflare 账号，请在页面右上角选择当前要使用的账号。"
        />
      </n-card>
    </template>
    <template v-else>
      <page-header
        title="Cloudflare DNS"
        description="在选定的 Zone 下管理 DNS 记录，可搜索和分页浏览。"
      >
        <template #actions>
          <n-space>
            <n-select
              v-model:value="currentZoneId"
              :options="zoneOptions"
              placeholder="选择 Zone"
              style="width: 220px"
            />
            <n-input
              v-model:value="query"
              placeholder="按 name / type / content 搜索"
              style="width: 260px"
              clearable
            />
            <n-select
              v-model:value="pageSize"
              :options="pageSizeOptions"
              style="width: 140px"
            />
            <n-button
              type="primary"
              :disabled="!currentZoneId"
              @click="showCreate = true"
            >
              新增记录
            </n-button>
            <n-button :disabled="!currentZoneId" :loading="loading" @click="loadRecords">获取最新</n-button>
          </n-space>
        </template>
      </page-header>
      <n-card class="page-section page-card">
        <div v-if="filteredRecords.length" class="cloudflare-stats">
          共 {{ records.length }} 条记录，当前筛选出 {{ filteredRecords.length }} 条。
        </div>
        <n-spin :show="loading">
          <n-data-table
            :columns="recordColumns"
            :data="pagedRecords"
            :bordered="false"
            :single-line="false"
            size="small"
          >
            <template #empty>
              <empty-state
                title="当前 Zone 下暂无 DNS 记录"
                description="可以为应用创建 A / AAAA / CNAME 等记录，或切换到其他 Zone。"
              />
            </template>
          </n-data-table>
        </n-spin>
        <div class="cloudflare-pagination">
          <n-pagination
            v-model:page="page"
            :page-size="pageSize"
            :item-count="filteredRecords.length"
            :page-sizes="[10, 20, 50, 100]"
            show-size-picker
            @update:page-size="onPageSizeChange"
          />
        </div>
      </n-card>

      <n-modal v-model:show="showCreate" title="新增 DNS 记录" preset="dialog" positive-text="创建" :loading="createLoading" @positive-click="onCreate">
      <n-form :model="createForm" label-placement="left" label-width="90" style="padding: 16px 0">
        <n-form-item label="类型" required>
          <n-select v-model:value="createForm.type" :options="typeOptions" style="width: 160px" />
        </n-form-item>
        <n-form-item label="名称" required>
          <n-input v-model:value="createForm.name" placeholder="例如：www 或 www.example.com" />
        </n-form-item>
        <n-form-item label="内容" required>
          <n-input v-model:value="createForm.content" placeholder="A: IP；CNAME: 目标域名" />
        </n-form-item>
        <n-form-item label="TTL">
          <n-input-number v-model:value="createForm.ttl" :min="0" placeholder="留空/0 自动" style="width: 180px" />
        </n-form-item>
        <n-form-item label="Proxied">
          <n-switch v-model:value="createForm.proxied" />
        </n-form-item>
      </n-form>
      </n-modal>

      <n-modal v-model:show="showEdit" title="编辑 DNS 记录" preset="dialog" positive-text="保存" :loading="editLoading" @positive-click="onEdit">
      <n-form :model="editForm" label-placement="left" label-width="90" style="padding: 16px 0">
        <n-form-item label="类型" required>
          <n-select v-model:value="editForm.type" :options="typeOptions" style="width: 160px" />
        </n-form-item>
        <n-form-item label="名称" required>
          <n-input v-model:value="editForm.name" />
        </n-form-item>
        <n-form-item label="内容" required>
          <n-input v-model:value="editForm.content" />
        </n-form-item>
        <n-form-item label="TTL">
          <n-input-number v-model:value="editForm.ttl" :min="0" placeholder="留空/0 自动" style="width: 180px" />
        </n-form-item>
        <n-form-item label="Proxied">
          <n-switch v-model:value="editForm.proxied" />
        </n-form-item>
      </n-form>
      </n-modal>
    </template>

    <n-modal
      v-model:show="showAccountsManager"
      title="Cloudflare 账号管理"
      preset="dialog"
      style="width: 720px"
    >
      <n-space vertical>
        <n-space align="center">
          <n-button size="small" :loading="accountsLoading" @click="loadAccounts">刷新</n-button>
          <n-button size="small" type="primary" @click="openCreateAccount">新增账号</n-button>
        </n-space>
        <n-data-table :columns="accountColumns" :data="accounts" :bordered="false" size="small" />
      </n-space>
    </n-modal>

    <cf-account-form-modal
      v-model:show="showAccountModal"
      :editing-account="editingAccount"
      @saved="onAccountSaved"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, h, inject } from 'vue'
import { useRoute } from 'vue-router'
import {
  NSelect,
  NButton,
  NDataTable,
  NSpace,
  NInput,
  NPagination,
  NSwitch,
  NModal,
  NForm,
  NFormItem,
  NInputNumber,
  NPopconfirm,
  NSpin,
  NEllipsis,
  NCard,
} from 'naive-ui'
import api from '../api/client'
import { notifySuccess } from '../ui/notify'
import { getCachedRecords, setCachedRecords } from '../utils/cfRecordsCache'
import PageHeader from '../components/PageHeader.vue'
import EmptyState from '../components/EmptyState.vue'
import CfAccountFormModal from '../components/CfAccountFormModal.vue'

const route = useRoute()

const cfCurrentAccountId = inject('cfCurrentAccountId', ref(null))
const cfAccounts = inject('cfAccounts', ref([]))
const refreshCfState = inject('refreshCfState', () => Promise.resolve())

const zones = ref([])
const records = ref([])
const currentZoneId = ref(null)
const loading = ref(false)

const accountsLoading = ref(false)
const accounts = ref([])
const showAccountModal = ref(false)
const editingAccount = ref(null)
const showAccountsManager = ref(false)

const zoneStorageKey = 'skylink_cf_zone_id'

const query = ref('')
const page = ref(1)
const pageSize = ref(20)
const pageSizeOptions = [
  { label: '10 / 页', value: 10 },
  { label: '20 / 页', value: 20 },
  { label: '50 / 页', value: 50 },
  { label: '100 / 页', value: 100 },
]

const zoneOptions = computed(() =>
  zones.value.map((z) => ({ label: z.name, value: z.id }))
)

const hasAnyCfAccount = computed(() => (cfAccounts.value || []).length > 0)
const needCfAccountSetup = computed(
  () => !hasAnyCfAccount.value || !cfCurrentAccountId.value
)

const typeOptions = [
  { label: 'A', value: 'A' },
  { label: 'CNAME', value: 'CNAME' },
  { label: 'TXT', value: 'TXT' },
  { label: 'AAAA', value: 'AAAA' },
  { label: 'MX', value: 'MX' },
]

const proxiedLoading = ref(new Set())

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

const recordColumns = computed(() => [
  { title: '类型', key: 'type', width: 90 },
  {
    title: '名称',
    key: 'name',
    ellipsis: true,
    render: (row) =>
      h(
        NEllipsis,
        { style: 'max-width: 260px' },
        { default: () => row.name }
      ),
  },
  {
    title: '内容',
    key: 'content',
    ellipsis: true,
    render: (row) =>
      h(
        NEllipsis,
        { style: 'max-width: 320px' },
        { default: () => row.content }
      ),
  },
  { title: 'TTL', key: 'ttl', width: 90 },
  {
    title: 'Proxied',
    key: 'proxied',
    width: 110,
    render: (row) =>
      h(NSwitch, {
        value: !!row.proxied,
        loading: proxiedLoading.value.has(row.id),
        onUpdateValue: (v) => toggleProxied(row, v),
      }),
  },
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
              onClick: () => openEdit(row),
            },
            { default: () => '编辑' }
          ),
          h(
            NPopconfirm,
            {
              positiveText: '删除',
              negativeText: '取消',
              onPositiveClick: () => onDelete(row),
            },
            {
              trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => '删除' }),
              default: () => '确定删除该记录？',
            }
          ),
        ],
      }),
  },
])

const filteredRecords = computed(() => {
  const q = (query.value || '').trim().toLowerCase()
  if (!q) return records.value
  return (records.value || []).filter((r) => {
    const name = (r?.name || '').toString().toLowerCase()
    const type = (r?.type || '').toString().toLowerCase()
    const content = (r?.content || '').toString().toLowerCase()
    return name.includes(q) || type.includes(q) || content.includes(q)
  })
})

const pagedRecords = computed(() => {
  const total = filteredRecords.value.length
  const ps = pageSize.value || 20
  const maxPage = Math.max(1, Math.ceil(total / ps))
  if (page.value > maxPage) page.value = 1
  const start = (page.value - 1) * ps
  return filteredRecords.value.slice(start, start + ps)
})

watch([query, pageSize], () => {
  page.value = 1
})

async function loadAccounts() {
  accountsLoading.value = true
  try {
    const { data } = await api.get('/cf/accounts')
    accounts.value = data.accounts || []
  } finally {
    accountsLoading.value = false
  }
}

async function loadZones() {
  try {
    const { data } = await api.get('/cf/zones')
    zones.value = data.zones || []
    const remembered = localStorage.getItem(zoneStorageKey)
    if (remembered && zones.value.some((z) => z.id === remembered)) {
      currentZoneId.value = remembered
    } else if (zones.value.length) {
      currentZoneId.value = zones.value[0].id
    }
  } catch (_) {}
}

function loadRecordsFromCache() {
  if (!currentZoneId.value) return
  const cached = getCachedRecords(cfCurrentAccountId.value, currentZoneId.value)
  records.value = cached != null ? cached : []
}

async function loadRecords() {
  if (!currentZoneId.value) return
  loading.value = true
  try {
    const { data } = await api.get(`/cf/zones/${currentZoneId.value}/records`)
    const list = data.records || []
    records.value = list
    setCachedRecords(cfCurrentAccountId.value, currentZoneId.value, list)
  } catch (_) {
    records.value = []
  } finally {
    loading.value = false
  }
}

watch(
  currentZoneId,
  (v) => {
    if (!v) return
    localStorage.setItem(zoneStorageKey, v)
    page.value = 1
    query.value = ''
    loadRecordsFromCache()
  },
  { immediate: false }
)

function onPageSizeChange(ps) {
  pageSize.value = ps
  page.value = 1
}

const showCreate = ref(false)
const createLoading = ref(false)
const createForm = ref({ type: 'CNAME', name: '', content: '', ttl: 1, proxied: true })

async function onCreate() {
  if (!currentZoneId.value) return false
  if (!createForm.value.name?.trim() || !createForm.value.content?.trim() || !createForm.value.type) return false
  createLoading.value = true
  try {
    await api.post(`/cf/zones/${currentZoneId.value}/records`, {
      type: createForm.value.type,
      name: createForm.value.name.trim(),
      content: createForm.value.content.trim(),
      ttl: normalizeTTL(createForm.value.ttl),
      proxied: !!createForm.value.proxied,
    })
    notifySuccess('创建成功', 'DNS 记录已创建')
    showCreate.value = false
    createForm.value = { type: 'CNAME', name: '', content: '', ttl: 1, proxied: true }
    await loadRecords()
    return true
  } finally {
    createLoading.value = false
  }
}

const showEdit = ref(false)
const editLoading = ref(false)
const editForm = ref({ id: '', type: 'CNAME', name: '', content: '', ttl: 1, proxied: true })

function openEdit(row) {
  editForm.value = {
    id: row.id,
    type: row.type,
    name: row.name,
    content: row.content,
    ttl: row.ttl,
    proxied: !!row.proxied,
  }
  showEdit.value = true
}

async function onEdit() {
  if (!currentZoneId.value) return false
  if (!editForm.value.id) return false
  if (!editForm.value.name?.trim() || !editForm.value.content?.trim() || !editForm.value.type) return false
  editLoading.value = true
  try {
    await api.put(`/cf/zones/${currentZoneId.value}/records/${editForm.value.id}`, {
      type: editForm.value.type,
      name: editForm.value.name.trim(),
      content: editForm.value.content.trim(),
      ttl: normalizeTTL(editForm.value.ttl),
      proxied: !!editForm.value.proxied,
    })
    notifySuccess('保存成功', 'DNS 记录已更新')
    showEdit.value = false
    await loadRecords()
    return true
  } finally {
    editLoading.value = false
  }
}

async function onDelete(row) {
  if (!currentZoneId.value) return
  await api.delete(`/cf/zones/${currentZoneId.value}/records/${row.id}`)
  notifySuccess('删除成功', 'DNS 记录已删除')
  await loadRecords()
}

function openAccountsManager() {
  showAccountsManager.value = true
  loadAccounts()
}

function openCreateAccount() {
  editingAccount.value = null
  showAccountModal.value = true
}

function openEditAccount(row) {
  editingAccount.value = row
  showAccountModal.value = true
}

async function deleteAccount(id) {
  await api.delete(`/cf/accounts/${id}`)
  await refreshCfState()
  await loadAccounts()
}

async function onAccountSaved() {
  showAccountModal.value = false
  await refreshCfState()
  await loadAccounts()
  if (!needCfAccountSetup.value) {
    await loadZones()
    loadRecordsFromCache()
  }
}

async function toggleProxied(row, v) {
  if (!currentZoneId.value) return
  if (proxiedLoading.value.has(row.id)) return
  const prev = !!row.proxied
  row.proxied = !!v
  proxiedLoading.value.add(row.id)
  proxiedLoading.value = new Set(proxiedLoading.value)
  try {
    await api.put(`/cf/zones/${currentZoneId.value}/records/${row.id}`, {
      type: row.type,
      name: row.name,
      content: row.content,
      ttl: normalizeTTL(row.ttl),
      proxied: !!v,
    })
    notifySuccess('已更新', 'proxied 状态已同步')
  } catch (_) {
    row.proxied = prev
  } finally {
    proxiedLoading.value.delete(row.id)
    proxiedLoading.value = new Set(proxiedLoading.value)
  }
}

function normalizeTTL(v) {
  const n = Number(v)
  if (Number.isNaN(n)) return 1
  if (!Number.isFinite(n)) return 1
  return Math.trunc(n)
}

onMounted(async () => {
  await loadAccounts()
  if (!needCfAccountSetup.value) {
    await loadZones()
    loadRecordsFromCache()
  }
  if (route.query.manage) {
    openAccountsManager()
  }
})

watch(
  () => route.query.manage,
  (val, oldVal) => {
    if (val && val !== oldVal) {
      openAccountsManager()
    }
  }
)
</script>
