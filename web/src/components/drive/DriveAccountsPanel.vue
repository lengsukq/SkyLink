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
import { computed, h } from 'vue'
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
import { formatBytes, formatBytesAsG } from '../../utils/storage'
import { type DriveAccount, useDriveAccountsCrud } from '../../composables/drive/useDriveAccountsCrud'

const {
  loading,
  creating,
  savingEdit,
  resettingPwd,
  list,
  showCreate,
  createForm,
  showEdit,
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
} = useDriveAccountsCrud()

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

</script>

<style scoped>
.token-alert {
  margin-bottom: 12px;
}
</style>
