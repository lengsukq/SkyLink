<template>
  <div class="layout">
    <div class="sidebar">
      <n-menu :value="activeCategory" :options="categoryOptions" @update:value="onCategory" />
    </div>

    <div class="main">
      <n-space vertical size="large">
        <div class="toolbar">
          <n-space align="center" wrap>
            <n-input v-model:value="path" placeholder="路径（相对根目录，例如：docs/）" style="min-width: 240px" />
            <n-input v-model:value="q" placeholder="文件名搜索（可选）" style="min-width: 200px" />
            <n-space align="center" :size="8">
              <n-select v-model:value="sort" :options="sortOptions" placeholder="排序" style="width: 160px" />
              <n-select v-model:value="order" :options="orderOptions" placeholder="顺序" style="width: 120px" />
              <n-switch v-model:value="recursive" />
              <span class="hint">递归</span>
            </n-space>
            <n-space align="center" :size="8">
              <n-button type="primary" :loading="loading" @click="refresh(true)">查询</n-button>
              <n-button secondary @click="openMkdir">新建文件夹</n-button>
              <n-button secondary :disabled="!canUpload" @click="pickUpload">上传</n-button>
              <input ref="fileInput" type="file" class="hidden" multiple @change="onFilePicked" />
            </n-space>
            <n-space align="center" :size="8">
              <n-button secondary :disabled="!selectedPaths.length" @click="batchDelete">删除</n-button>
              <n-button secondary :disabled="!clipboardCut" @click="pasteCut">粘贴</n-button>
              <n-button secondary @click="logout">退出</n-button>
            </n-space>
          </n-space>
        </div>

        <n-space align="center" justify="space-between">
          <div class="crumbs">
            <n-breadcrumb v-if="!recursive">
              <n-breadcrumb-item @click="goTo('')">根目录</n-breadcrumb-item>
              <n-breadcrumb-item v-for="seg in breadcrumb" :key="seg.path" @click="goTo(seg.path)">
                {{ seg.name }}
              </n-breadcrumb-item>
            </n-breadcrumb>
          </div>
          <n-space>
            <n-button secondary :disabled="!canLoadMore" @click="refresh(false)">加载更多</n-button>
          </n-space>
        </n-space>

        <n-card v-if="queue.length" size="small" class="uploader">
          <n-space vertical>
            <div class="uploader-title">上传队列</div>
            <n-space v-for="item in queue" :key="item.id" justify="space-between" align="center">
              <div class="uploader-name">{{ item.name }}</div>
              <div class="uploader-right">
                <n-progress type="line" :percentage="item.percent" :processing="item.status === 'uploading'" :indicator-placement="'inside'" />
              </div>
            </n-space>
          </n-space>
        </n-card>

        <div class="content">
          <div class="table">
            <n-space align="center" justify="space-between" class="view-toggle">
              <div class="view-title">文件</div>
              <n-space :size="8">
                <n-button size="small" secondary :type="viewMode === 'list' ? 'primary' : 'default'" @click="viewMode = 'list'">
                  列表
                </n-button>
                <n-button size="small" secondary :type="viewMode === 'grid' ? 'primary' : 'default'" @click="viewMode = 'grid'">
                  网格
                </n-button>
              </n-space>
            </n-space>

            <drive-grid-view
              v-if="viewMode === 'grid'"
              :items="rows"
              @open="openOrEnter"
              @select="(item) => (selectedPaths = [item.path])"
              @context="onGridContext"
            />

            <n-data-table
              v-else
              :columns="columns"
              :data="rows"
              :bordered="false"
              size="small"
              :single-line="false"
              :loading="loading"
              :row-key="rowKey"
              :checked-row-keys="selectedPaths"
              :row-props="rowProps"
              @update:checked-row-keys="onChecked"
            />
          </div>
          <div class="details">
            <drive-details-sidebar
              :item="selectedItem"
              @preview="selectedItem && openPreview(selectedItem)"
              @download="selectedItem && download(selectedItem)"
              @delete="selectedItem && remove(selectedItem)"
              @rename="selectedItem && openRename(selectedItem)"
              @cut="selectedItem && cut(selectedItem)"
              @copy-path="selectedItem && copyPath(selectedItem)"
            />
          </div>
        </div>
      </n-space>
    </div>
  </div>

  <drive-preview-modal
    v-model="previewOpen"
    :path="previewItem?.path || ''"
    :name="previewItem?.name || ''"
    :kind="previewKind"
    :size-bytes="previewItem?.size_bytes || 0"
  />

  <n-dropdown
    placement="bottom-start"
    trigger="manual"
    :x="ctx.x"
    :y="ctx.y"
    :options="ctxOptions"
    :show="ctx.show"
    @select="onCtxSelect"
    @clickoutside="ctx.show = false"
  />

  <n-modal v-model:show="renameModal" preset="card" title="重命名" style="max-width: 520px">
    <n-input v-model:value="renameValue" placeholder="新名称" />
    <template #footer>
      <n-space justify="end">
        <n-button @click="renameModal = false">取消</n-button>
        <n-button type="primary" :loading="renaming" @click="confirmRename">确定</n-button>
      </n-space>
    </template>
  </n-modal>

  <n-modal v-model:show="mkdirModal" preset="card" title="新建文件夹" style="max-width: 520px">
    <n-input v-model:value="mkdirValue" placeholder="文件夹名称" />
    <template #footer>
      <n-space justify="end">
        <n-button @click="mkdirModal = false">取消</n-button>
        <n-button type="primary" :loading="mkdiring" @click="confirmMkdir">确定</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, h, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  NBreadcrumb,
  NBreadcrumbItem,
  NButton,
  NCard,
  NDataTable,
  NDropdown,
  NInput,
  NMenu,
  NModal,
  NProgress,
  NSelect,
  NSpace,
  NSwitch,
} from 'naive-ui'
import { STORAGE_KEYS } from '../../constants/storage'
import { ROUTE_PATHS } from '../../constants/routes'
import { notifyError, notifySuccess } from '../../ui/notify'
import { copyToClipboard } from '../../utils/clipboard'
import { formatBytes } from '../../utils/storage'
import { driveUserDelete, driveUserDownloadBlob, driveUserListEntries, driveUserMkdir, driveUserRename, driveUserUpload } from '../../api/driveUserClient'
import DrivePreviewModal from './DrivePreviewModal.vue'
import DriveDetailsSidebar from './fm/DriveDetailsSidebar.vue'
import DriveGridView from './fm/DriveGridView.vue'

type DriveFileItem = {
  name: string
  path: string
  is_dir: boolean
  size_bytes: number
  modified_at: number
  ext: string
  type: string
}

const router = useRouter()

const path = ref('')
const q = ref('')
const activeCategory = ref<'all' | 'image' | 'video' | 'audio' | 'document' | 'archive' | 'other'>('all')
const recursive = ref(false)
const sort = ref<'name' | 'size' | 'mtime'>('name')
const order = ref<'asc' | 'desc'>('asc')

const loading = ref(false)
const cursor = ref('')
const hasMore = ref(false)
const rows = ref<DriveFileItem[]>([])
const selectedPaths = ref<string[]>([])
const previewOpen = ref(false)
const previewItem = ref<DriveFileItem | null>(null)

const fileInput = ref<HTMLInputElement | null>(null)

const categoryOptions = [
  { label: '全部', key: 'all' },
  { label: '图片', key: 'image' },
  { label: '视频', key: 'video' },
  { label: '音频', key: 'audio' },
  { label: '文档', key: 'document' },
  { label: '压缩包', key: 'archive' },
  { label: '其它', key: 'other' },
]

const sortOptions = [
  { label: '名称', value: 'name' },
  { label: '大小', value: 'size' },
  { label: '修改时间', value: 'mtime' },
]

const orderOptions = [
  { label: '升序', value: 'asc' },
  { label: '降序', value: 'desc' },
]

const breadcrumb = computed(() => {
  const p = (path.value || '').trim().replaceAll('\\', '/').replace(/^\/+/, '').replace(/\/+$/, '')
  if (!p) return []
  const parts = p.split('/').filter(Boolean)
  const out: Array<{ name: string; path: string }> = []
  let acc = ''
  for (const part of parts) {
    acc = acc ? `${acc}/${part}` : part
    out.push({ name: part, path: acc })
  }
  return out
})

const canLoadMore = computed(() => hasMore.value && !loading.value)
const canUpload = computed(() => !!(localStorage.getItem(STORAGE_KEYS.driveUserToken) || '').trim() && !loading.value)

const rowKey = (row: DriveFileItem) => row.path

const selectedItem = computed(() => {
  const key = selectedPaths.value?.[0]
  if (!key) return null
  return rows.value.find((r) => r.path === key) || null
})

const viewMode = ref<'list' | 'grid'>('list')

const columns = computed(() => [
  { type: 'selection' as const, width: 40 },
  {
    title: '名称',
    key: 'name',
    minWidth: 240,
    render(row: DriveFileItem) {
      const clickable = !recursive.value && row.is_dir
      return h(
        'span',
        {
          class: clickable ? 'link' : '',
          onClick: () => {
            if (clickable) goTo(row.path)
          },
        },
        row.name,
      )
    },
  },
  { title: '路径', key: 'path', minWidth: 260 },
  {
    title: '类型',
    key: 'type',
    width: 100,
    render(row: DriveFileItem) {
      return row.is_dir ? '目录' : row.type
    },
  },
  {
    title: '大小',
    key: 'size',
    width: 120,
    render(row: DriveFileItem) {
      return row.is_dir ? '-' : formatBytes(row.size_bytes)
    },
  },
  {
    title: '操作',
    key: 'actions',
    width: 240,
    render(row: DriveFileItem) {
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
                disabled: row.is_dir,
                onClick: () => openPreview(row),
              },
              { default: () => '预览' },
            ),
            h(
              NButton,
              {
                size: 'small',
                secondary: true,
                disabled: row.is_dir,
                onClick: () => download(row),
              },
              { default: () => '下载' },
            ),
            h(
              NButton,
              {
                size: 'small',
                tertiary: true,
                type: 'error',
                onClick: () => remove(row),
              },
              { default: () => '删除' },
            ),
          ],
        },
      )
    },
  },
])

function rowProps(row: DriveFileItem) {
  return {
    onDblclick: () => {
      if (row.is_dir) goTo(row.path)
      else openPreview(row)
    },
    onContextmenu: (e: MouseEvent) => {
      e.preventDefault()
      ctx.value.item = row
      ctx.value.x = e.clientX
      ctx.value.y = e.clientY
      ctx.value.show = true
      selectedPaths.value = [row.path]
    },
    onClick: () => {
      // Keep selection in sync with details panel.
      selectedPaths.value = [row.path]
    },
  }
}

async function refresh(reset: boolean) {
  loading.value = true
  try {
    if (reset) {
      cursor.value = ''
      rows.value = []
      selectedPaths.value = []
    }
    const type = activeCategory.value === 'all' ? '' : activeCategory.value
    const res = await driveUserListEntries({
      parent_path: recursive.value ? '' : path.value,
      path_prefix: recursive.value ? path.value : '',
      recursive: recursive.value,
      type,
      q: q.value || '',
      sort: sort.value,
      order: order.value,
      cursor: cursor.value || '',
      limit: 200,
      include_dirs: !type,
    })
    const incoming = (res.list || []) as DriveFileItem[]
    rows.value = reset ? incoming : rows.value.concat(incoming)
    cursor.value = String(res.next_cursor || '')
    hasMore.value = !!cursor.value
  } catch (e: any) {
    notifyError('查询失败', e?.response?.data?.error || e?.message || String(e))
  } finally {
    loading.value = false
  }
}

function goTo(p: string) {
  path.value = p
  q.value = ''
  cursor.value = ''
  hasMore.value = false
  rows.value = []
  selectedPaths.value = []
}

function pickUpload() {
  if (!fileInput.value) return
  fileInput.value.value = ''
  fileInput.value.click()
}

async function onFilePicked(e: Event) {
  const input = e.target as HTMLInputElement
  const files = Array.from(input.files || [])
  if (!files.length) return
  enqueue(files)
}

async function download(row: DriveFileItem) {
  try {
    const blob = await driveUserDownloadBlob(row.path)
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = row.name
    a.click()
    URL.revokeObjectURL(url)
  } catch (e: any) {
    notifyError('下载失败', e?.response?.data?.error || e?.message || String(e))
  }
}

async function remove(row: DriveFileItem) {
  try {
    await driveUserDelete(row.path)
    notifySuccess('已删除', row.name)
    await refresh(true)
  } catch (e: any) {
    notifyError('删除失败', e?.response?.data?.error || e?.message || String(e))
  }
}

function onCategory(v: any) {
  activeCategory.value = v
  refresh(true)
}

function onChecked(keys: any) {
  selectedPaths.value = (keys || []) as string[]
}

async function batchDelete() {
  const targets = [...selectedPaths.value]
  if (!targets.length) return
  try {
    for (const p of targets) {
      await driveUserDelete(p)
    }
    notifySuccess('已删除', `共 ${targets.length} 项`)
    await refresh(true)
  } catch (e: any) {
    notifyError('批量删除失败', e?.response?.data?.error || e?.message || String(e))
  }
}

type QueueItem = {
  id: string
  name: string
  file: File
  percent: number
  status: 'queued' | 'uploading' | 'done' | 'error'
}

const queue = ref<QueueItem[]>([])
const uploading = ref(0)
const MAX_CONCURRENCY = 3

function enqueue(files: File[]) {
  for (const f of files) {
    queue.value.push({
      id: `${Date.now()}-${Math.random()}`,
      name: f.name,
      file: f,
      percent: 0,
      status: 'queued',
    })
  }
  pump()
}

function pump() {
  if (uploading.value >= MAX_CONCURRENCY) return
  const next = queue.value.find((x) => x.status === 'queued')
  if (!next) return
  uploading.value += 1
  next.status = 'uploading'
  driveUserUpload(path.value, next.file, {
    onProgress: (loaded, total) => {
      if (total && total > 0) next.percent = Math.min(100, Math.round((loaded / total) * 100))
    },
  })
    .then(() => {
      next.status = 'done'
      next.percent = 100
    })
    .catch(() => {
      next.status = 'error'
    })
    .finally(() => {
      uploading.value -= 1
      pump()
      refresh(true)
    })
}

function logout() {
  localStorage.removeItem(STORAGE_KEYS.driveUserToken)
  router.push(ROUTE_PATHS.driveLogin)
}

function openPreview(row: DriveFileItem) {
  if (row.is_dir) return
  previewItem.value = row
  previewOpen.value = true
}

function openOrEnter(item: DriveFileItem) {
  if (item.is_dir) {
    goTo(item.path)
    return
  }
  openPreview(item)
}

const previewKind = computed(() => {
  const r = previewItem.value
  if (!r) return 'unknown'
  if (r.type === 'image') return 'image'
  if (r.type === 'video') return 'video'
  if (r.type === 'audio') return 'audio'
  if (r.ext?.toLowerCase?.() === 'pdf') return 'pdf'
  return 'unknown'
})

const ctx = ref<{ show: boolean; x: number; y: number; item: DriveFileItem | null }>({
  show: false,
  x: 0,
  y: 0,
  item: null,
})

const ctxOptions = computed(() => {
  const item = ctx.value.item
  const isFile = !!item && !item.is_dir
  return [
    { label: '预览', key: 'preview', disabled: !isFile },
    { label: '下载', key: 'download', disabled: !isFile },
    { label: '复制路径', key: 'copy-path', disabled: !item },
    { label: '重命名', key: 'rename', disabled: !item },
    { label: '剪切', key: 'cut', disabled: !item },
    { label: '删除', key: 'delete', disabled: !item },
  ]
})

function onCtxSelect(key: string) {
  const item = ctx.value.item
  ctx.value.show = false
  if (!item) return
  if (key === 'preview') openPreview(item)
  if (key === 'download') download(item)
  if (key === 'delete') remove(item)
  if (key === 'rename') openRename(item)
  if (key === 'cut') cut(item)
  if (key === 'copy-path') copyPath(item)
}

function onGridContext(item: DriveFileItem, e: MouseEvent) {
  ctx.value.item = item
  ctx.value.x = e.clientX
  ctx.value.y = e.clientY
  ctx.value.show = true
  selectedPaths.value = [item.path]
}

function cut(item: DriveFileItem) {
  clipboardCut.value = item.path
  notifySuccess('已剪切', item.name)
}

const clipboardCut = ref<string>('')

async function pasteCut() {
  const from = clipboardCut.value
  if (!from) return
  const name = from.includes('/') ? from.split('/').pop() || '' : from
  const base = path.value ? path.value.replace(/\/+$/, '') : ''
  const to = base ? `${base}/${name}` : name
  try {
    await driveUserRename(from, to)
    clipboardCut.value = ''
    notifySuccess('已移动', name)
    await refresh(true)
  } catch (e: any) {
    notifyError('移动失败', e?.response?.data?.error || e?.message || String(e))
  }
}

async function copyPath(item: DriveFileItem) {
  try {
    await copyToClipboard(item.path)
    notifySuccess('已复制', '路径已复制到剪贴板')
  } catch (e: any) {
    notifyError('复制失败', e?.message || String(e))
  }
}

const renameModal = ref(false)
const renaming = ref(false)
const renameItem = ref<DriveFileItem | null>(null)
const renameValue = ref('')

function openRename(item: DriveFileItem) {
  renameItem.value = item
  renameValue.value = item.name
  renameModal.value = true
}

async function confirmRename() {
  const item = renameItem.value
  const nextName = renameValue.value.trim()
  if (!item || !nextName) return
  const parent = item.path.includes('/') ? item.path.split('/').slice(0, -1).join('/') : ''
  const to = parent ? `${parent}/${nextName}` : nextName
  renaming.value = true
  try {
    await driveUserRename(item.path, to)
    notifySuccess('已重命名', nextName)
    renameModal.value = false
    await refresh(true)
  } catch (e: any) {
    notifyError('重命名失败', e?.response?.data?.error || e?.message || String(e))
  } finally {
    renaming.value = false
  }
}

const mkdirModal = ref(false)
const mkdirValue = ref('')
const mkdiring = ref(false)

function openMkdir() {
  mkdirValue.value = ''
  mkdirModal.value = true
}

async function confirmMkdir() {
  const name = mkdirValue.value.trim()
  if (!name) return
  const target = path.value ? `${path.value.replace(/\/+$/, '')}/${name}` : name
  mkdiring.value = true
  try {
    await driveUserMkdir(target)
    notifySuccess('已创建', name)
    mkdirModal.value = false
    await refresh(true)
  } catch (e: any) {
    notifyError('创建失败', e?.response?.data?.error || e?.message || String(e))
  } finally {
    mkdiring.value = false
  }
}
</script>

<style scoped>
.layout {
  display: grid;
  grid-template-columns: 180px 1fr;
  gap: 16px;
}
.sidebar {
  position: sticky;
  top: 12px;
  align-self: start;
}
.main {
  min-width: 0;
}
.content {
  display: grid;
  grid-template-columns: 1fr 280px;
  gap: 12px;
  align-items: start;
}
.table {
  min-width: 0;
}
.details {
  min-width: 0;
}
.view-toggle {
  margin-bottom: 8px;
}
.view-title {
  font-weight: 600;
}
.toolbar {
  padding: 8px 0 2px;
}
.hidden {
  display: none;
}
.hint {
  opacity: 0.8;
  font-size: 12px;
}
.link {
  cursor: pointer;
  text-decoration: underline;
}
.uploader {
  margin-top: 6px;
}
.uploader-title {
  font-weight: 600;
  font-size: 13px;
}
.uploader-name {
  max-width: 360px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.uploader-right {
  width: 260px;
}
</style>

