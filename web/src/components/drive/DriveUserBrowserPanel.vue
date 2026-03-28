<template>
  <div class="layout">
    <div
      class="sidebar rounded-2xl border border-white/50 bg-white/55 p-2 shadow-sm shadow-slate-900/5 backdrop-blur-xl"
    >
      <n-menu :value="activeCategory" :options="categoryOptions" @update:value="onCategory" />
    </div>

    <div class="main rounded-2xl border border-white/35 bg-white/45 p-4 shadow-sm shadow-slate-900/5 backdrop-blur-md md:p-5">
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
              <n-button type="primary" :loading="loading" @click="refreshWithToast(true)">查询</n-button>
              <n-tooltip trigger="hover">
                <template #trigger>
                  <n-button secondary :loading="reindexing" :disabled="!canUpload" @click="syncLocalIndex">
                    同步本地
                  </n-button>
                </template>
                重新扫描账号根目录并更新索引，使列表与磁盘上的真实文件一致（例如在服务器上直接增删文件后使用）
              </n-tooltip>
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
            <n-button secondary :disabled="!canLoadMore" @click="refreshWithToast(false)">加载更多</n-button>
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
    @clickoutside="closeCtx"
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
  NTooltip,
} from 'naive-ui'
import { STORAGE_KEYS } from '../../constants/storage'
import { ROUTE_PATHS } from '../../constants/routes'
import { notifyError, notifySuccess } from '../../ui/notify'
import { formatBytes } from '../../utils/storage'
import { DRIVE_CATEGORY_OPTIONS, DRIVE_ORDER_OPTIONS, DRIVE_SORT_OPTIONS } from '../../constants/drive'
import type { DriveEntry, DriveViewMode } from '../../types/drive'
import { useDriveQueryState } from '../../composables/drive/useDriveQueryState'
import { useDriveClipboardCutPaste } from '../../composables/drive/useDriveClipboardCutPaste'
import { useDriveContextMenu } from '../../composables/drive/useDriveContextMenu'
import { useDriveActions } from '../../composables/drive/useDriveActions'
import { useDriveModals } from '../../composables/drive/useDriveModals'
import { useDriveUploadQueue } from '../../composables/drive/useDriveUploadQueue'
import { driveUserIndexRebuild, driveUserIndexStatus } from '../../api/driveUserClient'
import DrivePreviewModal from './DrivePreviewModal.vue'
import DriveDetailsSidebar from './fm/DriveDetailsSidebar.vue'
import DriveGridView from './fm/DriveGridView.vue'

const router = useRouter()

const { path, q, activeCategory, recursive, sort, order, loading, rows, breadcrumb, canLoadMore, refresh, goTo } = useDriveQueryState()
const selectedPaths = ref<string[]>([])

const fileInput = ref<HTMLInputElement | null>(null)

const categoryOptions = DRIVE_CATEGORY_OPTIONS
const sortOptions = DRIVE_SORT_OPTIONS
const orderOptions = DRIVE_ORDER_OPTIONS
const canUpload = computed(() => !!(localStorage.getItem(STORAGE_KEYS.driveUserToken) || '').trim() && !loading.value)

const reindexing = ref(false)

const rowKey = (row: DriveEntry) => row.path

const selectedItem = computed(() => {
  const key = selectedPaths.value?.[0]
  if (!key) return null
  return rows.value.find((r) => r.path === key) || null
})

const viewMode = ref<DriveViewMode>('list')

const columns = computed(() => [
  { type: 'selection' as const, width: 40 },
  {
    title: '名称',
    key: 'name',
    minWidth: 240,
    render(row: DriveEntry) {
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
    render(row: DriveEntry) {
      return row.is_dir ? '目录' : row.type
    },
  },
  {
    title: '大小',
    key: 'size',
    width: 120,
    render(row: DriveEntry) {
      return row.is_dir ? '-' : formatBytes(row.size_bytes)
    },
  },
  {
    title: '操作',
    key: 'actions',
    width: 240,
    render(row: DriveEntry) {
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

function rowProps(row: DriveEntry) {
  return {
    onDblclick: () => {
      if (row.is_dir) goTo(row.path)
      else openPreview(row)
    },
    onContextmenu: (e: MouseEvent) => {
      e.preventDefault()
      openContextMenu(row, e)
    },
    onClick: () => {
      // Keep selection in sync with details panel.
      selectedPaths.value = [row.path]
    },
  }
}

async function refreshWithToast(reset: boolean) {
  try {
    if (reset) selectedPaths.value = []
    await refresh(reset)
  } catch (e: any) {
    notifyError('查询失败', e?.response?.data?.error || e?.message || String(e))
  }
}

const INDEX_POLL_MAX = 7200
const INDEX_POLL_MS = 500

async function syncLocalIndex() {
  reindexing.value = true
  try {
    try {
      await driveUserIndexRebuild()
    } catch (e: any) {
      const msg = String(e?.response?.data?.error ?? '')
      if (!msg.includes('already running')) throw e
    }
    let timedOut = false
    for (let i = 0; i < INDEX_POLL_MAX; i++) {
      const { status } = await driveUserIndexStatus()
      if (!status) break
      if (!status.running) {
        if (status.last_error) {
          notifyError('同步失败', status.last_error)
          return
        }
        break
      }
      if (i === INDEX_POLL_MAX - 1) {
        timedOut = true
        break
      }
      await new Promise((r) => setTimeout(r, INDEX_POLL_MS))
    }
    if (timedOut) {
      notifySuccess('提示', '同步任务仍在进行，请稍后点击「查询」刷新列表')
    } else {
      notifySuccess('同步完成', '列表已与本地目录对齐')
    }
    await refreshWithToast(true)
  } catch (e: any) {
    notifyError('同步失败', e?.response?.data?.error || e?.message || String(e))
  } finally {
    reindexing.value = false
  }
}

const {
  previewOpen,
  previewItem,
  previewKind,
  openPreview,
  openOrEnter: openOrEnterRaw,
  download,
  remove,
  batchDelete: batchDeleteRaw,
  copyPath: copyPathRaw,
} = useDriveActions({
  refresh: refreshWithToast,
  notifySuccess,
  notifyError,
})

function pickUpload() {
  if (!fileInput.value) return
  fileInput.value.value = ''
  fileInput.value.click()
}

async function onFilePicked(e: Event) {
  const input = e.target as HTMLInputElement
  const files = Array.from(input.files || [])
  if (!files.length) return
  enqueueFiles(files)
}

function onCategory(v: any) {
  activeCategory.value = v
  refreshWithToast(true)
}

function onChecked(keys: any) {
  selectedPaths.value = (keys || []) as string[]
}

async function batchDelete() {
  await batchDeleteRaw(selectedPaths.value)
}

const { queue, enqueueFiles } = useDriveUploadQueue({
  getTargetDir: () => path.value,
  onSettled: () => refreshWithToast(true),
})

function logout() {
  localStorage.removeItem(STORAGE_KEYS.driveUserToken)
  router.push(ROUTE_PATHS.driveLogin)
}

function openOrEnter(item: DriveEntry) {
  openOrEnterRaw(item, goTo)
}

const { ctx, options: ctxOptions, openAt: openContextMenu, onSelect: onCtxSelect, close: closeCtx } = useDriveContextMenu({
  onSelectItem: (item) => (selectedPaths.value = [item.path]),
  onAction: async (key, item) => {
    if (key === 'preview') openPreview(item)
    if (key === 'download') await download(item)
    if (key === 'delete') await remove(item)
    if (key === 'rename') openRename(item)
    if (key === 'cut') cut(item)
    if (key === 'copy-path') await copyPath(item)
  },
})

function onGridContext(item: DriveEntry, e: MouseEvent) {
  openContextMenu(item, e)
}

const { clipboardCutPath: clipboardCut, cut: cutPath, pasteCut: pasteCutRaw } = useDriveClipboardCutPaste({
  getCurrentDir: () => path.value,
  onMoved: async (name) => {
    notifySuccess('已移动', name)
    await refreshWithToast(true)
  },
})

const {
  renameModal,
  renaming,
  renameItem,
  renameValue,
  openRename,
  confirmRename: confirmRenameRaw,
  mkdirModal,
  mkdirValue,
  mkdiring,
  openMkdir,
  confirmMkdir: confirmMkdirRaw,
} = useDriveModals({
  getCurrentDir: () => path.value,
  onChanged: () => refreshWithToast(true),
})

function cut(item: DriveEntry) {
  cutPath(item.path)
  notifySuccess('已剪切', item.name)
}

async function pasteCut() {
  try {
    await pasteCutRaw()
  } catch (e: any) {
    notifyError('移动失败', e?.response?.data?.error || e?.message || String(e))
  }
}

async function copyPath(item: DriveEntry) {
  await copyPathRaw(item.path)
}

async function confirmRename() {
  try {
    const nextName = renameValue.value.trim()
    await confirmRenameRaw()
    if (nextName) notifySuccess('已重命名', nextName)
  } catch (e: any) {
    notifyError('重命名失败', e?.response?.data?.error || e?.message || String(e))
  }
}

async function confirmMkdir() {
  try {
    const name = mkdirValue.value.trim()
    await confirmMkdirRaw()
    if (name) notifySuccess('已创建', name)
  } catch (e: any) {
    notifyError('创建失败', e?.response?.data?.error || e?.message || String(e))
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

