<template>
  <div>
    <page-header
      title="映射管理"
      description="管理域名到后端服务的映射，可选配合 Cloudflare CNAME 一键创建。CF 状态基于 Cloudflare 页「获取最新」的缓存，若未同步请先到 Cloudflare 页拉取。"
    >
      <template #actions>
        <n-space wrap :size="8">
          <n-button type="primary" size="small" @click="showAdd = true">添加映射</n-button>
          <n-button size="small" @click="openOneClick">一键映射（含 CF CNAME）</n-button>
          <n-button size="small" @click="load">刷新</n-button>
        </n-space>
      </template>
    </page-header>
    <n-card class="page-section page-card">
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
    </n-card>

    <n-modal v-model:show="showAdd" title="添加映射" preset="dialog" positive-text="添加" @positive-click="onAdd">
      <n-form :model="addForm" label-placement="left" label-width="80" style="padding: 16px 0">
        <n-form-item label="域名" path="host" required>
          <n-input v-model:value="addForm.host" placeholder="xx.yourdomain.com" />
        </n-form-item>
        <n-form-item label="后端" path="backend" required>
          <n-input v-model:value="addForm.backend" placeholder="http://127.0.0.1:3000" />
        </n-form-item>
        <n-form-item label="从 mesh 选择">
          <n-space>
            <n-select
              v-model:value="meshSelectedIp"
              :options="meshIpOptions"
              placeholder="选择 mesh 节点 IP"
              clearable
              style="width: 160px"
            />
            <n-input-number v-model:value="meshPort" :min="1" :max="65535" placeholder="端口" style="width: 100px" />
            <n-button size="small" :disabled="!meshSelectedIp" @click="fillBackendFromMesh('add')">填入</n-button>
          </n-space>
        </n-form-item>
      </n-form>
    </n-modal>

    <n-modal
      v-model:show="showNoFrpHint"
      preset="dialog"
      title="需要设置 FRP 固定域名"
      positive-text="前往设置"
      negative-text="取消"
      @positive-click="goToSettings"
    >
      <template #default>
        一键映射会在 Cloudflare 中创建指向 FRP 固定域名的 CNAME 记录，请先前往「设置」页填写 FRP 固定域名后再使用一键映射。
      </template>
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
          <template #feedback>
            <span class="text-xs text-slate-500">
              <template v-if="frpCnameTarget">
                <template v-if="!oneClickForm.cname_target?.trim()">
                  未填写时将使用设置页中的默认 FRP 固定域名：{{ frpCnameTarget }}。
                </template>
                <template v-else-if="oneClickForm.cname_target?.trim() === frpCnameTarget">
                  当前将使用设置页中的默认 FRP 固定域名：{{ frpCnameTarget }}。
                </template>
                <template v-else>
                  当前将覆盖设置页中的默认 FRP 固定域名。
                </template>
              </template>
              <template v-else>
                尚未在设置页配置 FRP 固定域名，将无法自动创建指向 FRP 的 Cloudflare CNAME。
              </template>
            </span>
          </template>
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
        <n-form-item label="从 mesh 选择">
          <n-space>
            <n-select
              v-model:value="meshSelectedIpEdit"
              :options="meshIpOptions"
              placeholder="选择 mesh 节点 IP"
              clearable
              style="width: 160px"
            />
            <n-input-number v-model:value="meshPortEdit" :min="1" :max="65535" placeholder="端口" style="width: 100px" />
            <n-button size="small" :disabled="!meshSelectedIpEdit" @click="fillBackendFromMesh('edit')">填入</n-button>
          </n-space>
        </n-form-item>
      </n-form>
    </n-modal>

    <n-modal
      v-model:show="showNoCfHint"
      preset="dialog"
      title="需要 Cloudflare 账号"
      positive-text="去添加"
      @positive-click="showNoCfHint = false; showAddAccountModal = true"
    >
      <template #default>
        请先在页面右上角添加并选择 Cloudflare 账号，才能使用一键映射创建 CNAME。
      </template>
    </n-modal>

    <cf-account-form-modal
      v-model:show="showAddAccountModal"
      :editing-account="null"
      @saved="onCfAccountSaved"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, h, onMounted, computed, inject, watch } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NPopconfirm, NSpace, NDataTable, NModal, NForm, NFormItem, NInput, NInputNumber, NSelect, NSpin, NEllipsis, NTag, NCard } from 'naive-ui'
import api from '../api/client'
import { getAllCachedRecordsForAccount, setCachedRecords } from '../utils/cfRecordsCache'
import PageHeader from '../components/PageHeader.vue'
import EmptyState from '../components/EmptyState.vue'
import CfAccountFormModal from '../components/CfAccountFormModal.vue'
import { cfCurrentAccountIdKey, refreshCfStateKey } from '../types/cfContext'

const router = useRouter()

type Zone = { id: number | string; name: string }
type MeshIpOption = { label: string; value: string }
type FrpSettings = { frp_cname_target: string }
type CfStatusType = 'default' | 'success' | 'warning'
type CfStatus = { text: string; type: CfStatusType }

const list = ref<any[]>([])
const loading = ref(false)
const showAdd = ref(false)
const showEdit = ref(false)
const showOneClick = ref(false)
const showNoCfHint = ref(false)
const showNoFrpHint = ref(false)
const showAddAccountModal = ref(false)
const addForm = ref<{ host: string; backend: string }>({ host: '', backend: '' })
const oneClickForm = ref<{ host: string; backend: string; zone_id: string; cname_target: string }>({
  host: '',
  backend: '',
  zone_id: '',
  cname_target: '',
})
const editForm = ref<{ id: number | null; host: string; backend: string }>({ id: null, host: '', backend: '' })

const zones = ref<Zone[]>([])
const cfCurrentAccountId = inject(cfCurrentAccountIdKey, ref<number | null>(null))
const refreshCfState = inject(refreshCfStateKey, () => Promise.resolve())
const zoneOptions = computed(() => zones.value.map((z) => ({ label: z.name, value: z.id })))

const meshIpOptions = ref<MeshIpOption[]>([])
const meshSelectedIp = ref<string | null>(null)
const meshPort = ref<number>(3000)
const meshSelectedIpEdit = ref<string | null>(null)
const meshPortEdit = ref<number>(3000)

const settings = ref<FrpSettings>({ frp_cname_target: '' })

const frpCnameTarget = computed(() => (settings.value.frp_cname_target || '').trim())

async function loadMeshIps() {
  try {
    const { data } = await api.get('/easytier/status/all')
    const ips: MeshIpOption[] = []
    ;(data?.profiles || []).forEach((item: any) => {
      const st = item?.status
      if (!item?.ok || !st) return
      if (st?.self_ipv4) ips.push({ label: `${st.self_ipv4} (${item?.name || item?.id || '本机'})`, value: st.self_ipv4 })
      ;(st?.peers || []).forEach((p: any) => {
        if (p.ipv4 && p.ipv4 !== st?.self_ipv4) ips.push({ label: `${p.ipv4} (${item?.name || item?.id || 'peer'})`, value: p.ipv4 })
      })
    })
    meshIpOptions.value = ips
  } catch (_) {
    meshIpOptions.value = []
  }
}

async function loadSettings() {
  try {
    const { data } = await api.get('/settings')
    settings.value = data?.frp_cname_target
      ? { frp_cname_target: String(data.frp_cname_target) }
      : { frp_cname_target: '' }
  } catch (_) {
    settings.value = { frp_cname_target: '' }
  }
}

function fillBackendFromMesh(which: 'add' | 'edit') {
  const ip = which === 'add' ? meshSelectedIp.value : meshSelectedIpEdit.value
  const port = which === 'add' ? meshPort.value : meshPortEdit.value
  if (!ip) return
  const url = `http://${ip}:${port || 3000}`
  if (which === 'add') addForm.value.backend = url
  else editForm.value.backend = url
}

watch([showAdd, showEdit], ([add, edit]) => {
  if (add || edit) loadMeshIps()
})

function normalizeName(name: string) {
  return (name || '').trim().toLowerCase().replace(/\.$/, '')
}

const allCfRecords = computed(() => getAllCachedRecordsForAccount(cfCurrentAccountId.value))
const cfRecordNamesSet = computed(() => {
  const records = allCfRecords.value
  return new Set(records.map((r) => normalizeName(String(r.name))))
})

function getCfStatus(host: string): CfStatus {
  if (cfCurrentAccountId.value == null) return { text: '未同步', type: 'default' }
  if (allCfRecords.value.length === 0) return { text: '未同步', type: 'default' }
  return cfRecordNamesSet.value.has(normalizeName(host))
    ? { text: '已映射', type: 'success' }
    : { text: '未映射', type: 'warning' }
}

const columns = computed(() => [
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
    render(row: any) {
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
    render(row: any) {
      return h(
        NEllipsis,
        { style: 'max-width: 320px' },
        { default: () => row.backend }
      )
    },
  },
  {
    title: 'CF 状态',
    key: 'cfStatus',
    width: 100,
    render(row: any) {
      const status = getCfStatus(row.host)
      return h(NTag, { type: status.type, size: 'small' }, { default: () => status.text })
    },
  },
  {
    title: '操作',
    key: 'actions',
    width: 160,
    render(row: any) {
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
])

async function load() {
  loading.value = true
  try {
    const { data } = await api.get('/mappings')
    list.value = data?.list || []
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

function openOneClick() {
  if ((zones.value || []).length === 0) {
    showNoCfHint.value = true
    return
  }
  if (!frpCnameTarget.value) {
    showNoFrpHint.value = true
    return
  }
  showOneClick.value = true
}

function onCfAccountSaved() {
  refreshCfState().then(() => loadZones())
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
  const zoneId = oneClickForm.value.zone_id
  if (zoneId && cfCurrentAccountId.value != null) {
    try {
      const { data } = await api.get(`/cf/zones/${zoneId}/records`)
      const records = data?.records || []
      setCachedRecords(cfCurrentAccountId.value, zoneId, records)
    } catch (_) {}
  }
  await load()
  return true
}

function goToSettings() {
  showNoFrpHint.value = false
  router.push('/settings')
}

async function onEdit() {
  if (!editForm.value.backend?.trim()) return false
  await api.put(`/mappings/${editForm.value.id}`, { backend: editForm.value.backend })
  await load()
  return true
}

async function remove(id: number) {
  await api.delete(`/mappings/${id}`)
  await load()
}

onMounted(() => {
  load()
  loadZones()
  loadSettings()
})
</script>
