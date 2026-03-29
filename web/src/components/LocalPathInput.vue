<template>
  <div class="local-path-input">
    <n-input-group>
      <n-input
        :value="modelValue"
        :placeholder="placeholder"
        clearable
        @update:value="$emit('update:modelValue', $event)"
        @blur="onBlur"
      />
      <n-button type="primary" ghost @click="openBrowser">
        <span class="local-path-input__btn-label">浏览…</span>
      </n-button>
    </n-input-group>

    <n-collapse-transition :show="hintExpanded">
      <p class="local-path-input__hint">
        <strong>手动输入：</strong>填运行 SkyLink 的机器上的绝对路径（Windows 可用
        <code>C:\Users\名字</code> 或 <code>C:/Users/名字</code>）。
      </p>
    </n-collapse-transition>
    <n-button
      quaternary
      size="tiny"
      class="local-path-input__hint-toggle"
      @click="hintExpanded = !hintExpanded"
    >
      {{ hintExpanded ? '收起说明' : '路径填法说明' }}
    </n-button>

    <n-modal
      v-model:show="showBrowse"
      preset="card"
      :style="{ width: 'min(600px, 94vw)', maxHeight: 'min(640px, 92vh)' }"
      :mask-closable="false"
      :auto-focus="false"
      class="local-path-modal"
    >
      <template #header>
        <div class="local-path-modal__head">
          <div class="local-path-modal__title">在服务器上选择文件夹</div>
          <div class="local-path-modal__sub">
            列表显示的是 <strong>SkyLink 服务端</strong>能访问的磁盘路径，不是您浏览器所在电脑的路径。
          </div>
        </div>
      </template>

      <div class="local-path-modal__body">
        <n-alert type="info" :bordered="false" class="local-path-modal__tip">
          单击文件夹进入；选好位置后点右下角「使用此文件夹」。盘符列表请先进入某个盘再确认。
        </n-alert>

        <div class="local-path-modal__location">
          <div class="local-path-modal__location-label">当前位置</div>
          <div class="local-path-modal__location-path" :title="locationDisplay">
            {{ locationDisplay }}
          </div>
        </div>

        <div class="local-path-modal__toolbar">
          <n-button size="small" :disabled="!canGoUp || browseLoading" @click="goParent">
            ← 返回上一级
          </n-button>
          <n-button size="small" quaternary :disabled="browseLoading" @click="refreshListing">
            刷新
          </n-button>
        </div>

        <div class="local-path-modal__scroll-outer">
          <n-spin :show="browseLoading" class="local-path-modal__spin">
            <n-scrollbar class="local-path-modal__scroll" trigger="hover">
            <div v-if="browseEntries.length > 0" class="local-path-modal__list" role="list">
              <button
                v-for="e in browseEntries"
                :key="e.path"
                type="button"
                class="local-path-modal__item"
                role="listitem"
                @click="enterDir(e.path)"
              >
                <span class="local-path-modal__item-icon" aria-hidden="true">📁</span>
                <span class="local-path-modal__item-main">
                  <span class="local-path-modal__item-name">{{ e.name }}</span>
                  <span class="local-path-modal__item-path">{{ e.path }}</span>
                </span>
                <span class="local-path-modal__item-go" aria-hidden="true">进入</span>
              </button>
            </div>
            <n-empty
              v-else-if="!browseLoading"
              size="small"
              description="没有子文件夹"
              class="local-path-modal__empty"
            >
              <template #extra>
                <n-text depth="3" style="font-size: 13px">
                  若这就是目标目录，可直接点「使用此文件夹」。
                </n-text>
              </template>
            </n-empty>
            </n-scrollbar>
          </n-spin>
        </div>
      </div>

      <template #footer>
        <n-space justify="space-between" align="center" style="width: 100%">
          <n-text depth="3" style="font-size: 12px">将填入上方输入框并关闭窗口</n-text>
          <n-space :size="10">
            <n-button @click="showBrowse = false">取消</n-button>
            <n-button type="primary" :disabled="!canPickCurrent || browseLoading" @click="pickCurrent">
              使用此文件夹
            </n-button>
          </n-space>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  NAlert,
  NButton,
  NCollapseTransition,
  NEmpty,
  NInput,
  NInputGroup,
  NModal,
  NScrollbar,
  NSpace,
  NSpin,
  NText,
} from 'naive-ui'
import api from '../api/client'
import { notifyError } from '../ui/notify'
import { normalizeLocalPath } from '../utils/localPath'

const props = withDefaults(
  defineProps<{
    modelValue: string
    placeholder?: string
    /** 不传则从 /api/stats 推断 */
    isWindows?: boolean
  }>(),
  {
    placeholder: '',
    isWindows: undefined,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const isWin = ref(false)
const hintExpanded = ref(false)
const showBrowse = ref(false)
const browseLoading = ref(false)
const browseCurrent = ref('')
const browseParent = ref('')
const browseEntries = ref<{ name: string; path: string; is_dir: boolean }[]>([])

const locationDisplay = computed(() => {
  const c = browseCurrent.value
  if (c === '') return '（请选择磁盘或卷 — Windows：先点进某个盘符）'
  if (c === '/') return '/（根目录）'
  return c
})

const canPickCurrent = computed(() => browseCurrent.value !== '')

const canGoUp = computed(() => {
  const c = browseCurrent.value
  if (c === '' || c === '/') return false
  return true
})

async function resolveIsWindows() {
  if (props.isWindows !== undefined) {
    isWin.value = props.isWindows
    return
  }
  try {
    const { data } = await api.get('/stats')
    isWin.value = !!data?.is_windows
  } catch {
    isWin.value = false
  }
}

onMounted(() => {
  void resolveIsWindows()
})

function onBlur() {
  emit('update:modelValue', normalizeLocalPath(props.modelValue, isWin.value))
}

async function loadBrowse(path: string) {
  browseLoading.value = true
  try {
    const { data } = await api.get('/fs/browse', {
      params: path ? { path } : {},
      silentError: true,
    } as any)
    browseCurrent.value = data?.current ?? ''
    browseParent.value = data?.parent ?? ''
    browseEntries.value = Array.isArray(data?.entries) ? data.entries : []
  } catch (e: any) {
    browseEntries.value = []
    const msg = e?.response?.data?.error || e?.message || '无法列出目录'
    notifyError('无法打开目录', msg)
  } finally {
    browseLoading.value = false
  }
}

function refreshListing() {
  void loadBrowse(browseCurrent.value)
}

function openBrowser() {
  showBrowse.value = true
  const seed = props.modelValue.trim()
  void loadBrowse(seed.length >= 2 ? seed : '')
}

function enterDir(path: string) {
  void loadBrowse(path)
}

function goParent() {
  const p = browseParent.value
  if (p === '') {
    void loadBrowse('')
    return
  }
  void loadBrowse(p)
}

function pickCurrent() {
  const cur = browseCurrent.value
  if (cur === '') return
  emit('update:modelValue', normalizeLocalPath(cur, isWin.value))
  showBrowse.value = false
}
</script>

<style scoped>
.local-path-input {
  width: 100%;
}
.local-path-input__btn-label {
  font-weight: 500;
}
.local-path-input__hint {
  margin: 8px 0 0;
  font-size: 12px;
  line-height: 1.55;
  color: var(--n-text-color-3);
}
.local-path-input__hint code {
  font-size: 11px;
  padding: 1px 5px;
  border-radius: 4px;
  background: rgba(128, 128, 128, 0.12);
}
.local-path-input__hint-toggle {
  margin-top: 2px;
  padding-left: 0 !important;
  font-size: 12px;
}

.local-path-modal :deep(.n-card) {
  max-height: min(640px, 92vh);
  display: flex;
  flex-direction: column;
}
.local-path-modal :deep(.n-card__content) {
  flex: 1 1 auto;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.local-path-modal :deep(.n-card-header) {
  padding-bottom: 8px;
  flex-shrink: 0;
}
.local-path-modal__head {
  padding-right: 8px;
}
.local-path-modal__title {
  font-size: 17px;
  font-weight: 600;
  line-height: 1.35;
  color: var(--n-text-color-1);
}
.local-path-modal__sub {
  margin-top: 6px;
  font-size: 13px;
  line-height: 1.5;
  color: var(--n-text-color-3);
}
.local-path-modal__body {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow: hidden;
}
.local-path-modal__tip {
  font-size: 13px;
  line-height: 1.5;
}
.local-path-modal__location {
  padding: 10px 12px;
  border-radius: 8px;
  background: rgba(128, 128, 128, 0.08);
  border: 1px solid var(--n-border-color);
}
.local-path-modal__location-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--n-text-color-3);
  margin-bottom: 6px;
}
.local-path-modal__location-path {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 13px;
  line-height: 1.45;
  word-break: break-all;
  color: var(--n-text-color-1);
  max-height: 4.5em;
  overflow-y: auto;
}
.local-path-modal__toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
.local-path-modal__tip,
.local-path-modal__location {
  flex-shrink: 0;
}
.local-path-modal__scroll-outer {
  flex: 1 1 auto;
  min-height: 120px;
  max-height: min(420px, 48vh);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--n-border-color);
  border-radius: 8px;
  background: var(--n-color-modal);
}
.local-path-modal__spin {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.local-path-modal__spin :deep(.n-spin-container) {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.local-path-modal__spin :deep(.n-spin-content) {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.local-path-modal__scroll {
  flex: 1 1 auto;
  min-height: 0;
  height: 100%;
  max-height: none;
  border: none;
  border-radius: 0;
  background: transparent;
}
.local-path-modal__scroll :deep(.n-scrollbar) {
  height: 100%;
  max-height: 100%;
}
.local-path-modal__scroll :deep(.n-scrollbar-container) {
  max-height: 100%;
}
.local-path-modal__list {
  padding: 4px 0;
}
.local-path-modal__item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  width: 100%;
  margin: 0;
  padding: 10px 12px;
  border: none;
  border-bottom: 1px solid var(--n-divider-color);
  background: transparent;
  cursor: pointer;
  text-align: left;
  transition: background 0.15s ease;
}
.local-path-modal__item:last-child {
  border-bottom: none;
}
.local-path-modal__item:hover {
  background: rgba(128, 128, 128, 0.09);
}
.local-path-modal__item:focus-visible {
  outline: 2px solid var(--n-color-target);
  outline-offset: -2px;
}
.local-path-modal__item-icon {
  flex-shrink: 0;
  font-size: 18px;
  line-height: 1.2;
  opacity: 0.9;
}
.local-path-modal__item-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.local-path-modal__item-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--n-text-color-1);
}
.local-path-modal__item-path {
  font-size: 12px;
  color: var(--n-text-color-3);
  word-break: break-all;
  line-height: 1.35;
}
.local-path-modal__item-go {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--n-primary-color);
  padding-top: 2px;
}
.local-path-modal__empty {
  padding: 20px 16px 24px;
}
.local-path-modal :deep(.n-card__footer) {
  flex-shrink: 0;
  border-top: 1px solid var(--n-border-color);
}
</style>
