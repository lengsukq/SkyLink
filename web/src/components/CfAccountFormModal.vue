<template>
  <n-modal
    :show="show"
    preset="dialog"
    :title="editingAccount ? '编辑 Cloudflare 账号' : '新增 Cloudflare 账号'"
    positive-text="保存"
    :positive-button-props="{ disabled: !zoneValidated }"
    :loading="accountSaving"
    @update:show="(v) => $emit('update:show', v)"
    @positive-click="onSave"
  >
    <n-form :model="form" label-placement="left" label-width="120" style="padding: 16px 0">
      <n-form-item label="名称" required>
        <n-input v-model:value="form.name" placeholder="例如：个人账号 / 公司账号" />
      </n-form-item>
      <n-form-item label="API Token" :required="!editingAccount">
        <n-input
          v-model:value="form.api_token"
          type="password"
          show-password-on="click"
          placeholder="Cloudflare API Token（Zone.DNS Edit 权限）"
        />
      </n-form-item>
      <n-form-item label="默认 Zone">
        <n-select
          v-model:value="form.zone_id"
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
</template>

<script setup>
import { ref, reactive, computed, watch } from 'vue'
import { NModal, NForm, NFormItem, NInput, NSelect, NSpace, NButton } from 'naive-ui'
import api from '../api/client'
import { notifySuccess } from '../ui/notify'

const props = defineProps({
  show: { type: Boolean, default: false },
  editingAccount: { type: Object, default: null },
})

const emit = defineEmits(['update:show', 'saved'])

const accountSaving = ref(false)
const form = reactive({ name: '', api_token: '', zone_id: '' })
const zoneValidated = ref(false)
const zoneChecking = ref(false)
const availableZones = ref([])

const zoneOptions = computed(() =>
  (availableZones.value || []).map((z) => ({
    label: `${z.name} (${z.id})`,
    value: z.id,
  }))
)

watch(
  () => [props.show, props.editingAccount],
  ([show, editing]) => {
    if (show) {
      if (editing) {
        form.name = editing.name || ''
        form.api_token = ''
        form.zone_id = editing.zone_id || ''
        zoneValidated.value = true
        availableZones.value = []
      } else {
        form.name = ''
        form.api_token = ''
        form.zone_id = ''
        zoneValidated.value = false
        availableZones.value = []
      }
    }
  },
  { immediate: true }
)

async function validateZone() {
  if (!form.api_token.trim()) return
  zoneChecking.value = true
  try {
    const { data } = await api.post('/cf/accounts/validate', {
      api_token: form.api_token.trim(),
      zone_id: (form.zone_id || '').trim(),
    })
    availableZones.value = data.zones || []
    if (data?.zone_id) {
      form.zone_id = data.zone_id
    } else if (availableZones.value.length && !form.zone_id) {
      form.zone_id = availableZones.value[0].id
    }
    zoneValidated.value = true
    notifySuccess('验证通过', 'API Token 与 Zone 设置已通过校验')
  } finally {
    zoneChecking.value = false
  }
}

async function onSave() {
  if (!form.name.trim()) return false
  if (!props.editingAccount && !form.api_token.trim()) return false
  if (!zoneValidated.value) return false

  accountSaving.value = true
  try {
    if (props.editingAccount) {
      await api.put(`/cf/accounts/${props.editingAccount.id}`, {
        name: form.name.trim(),
        api_token: form.api_token.trim() || undefined,
        zone_id: form.zone_id.trim(),
      })
      emit('saved')
      emit('update:show', false)
      notifySuccess('保存成功', 'Cloudflare 账号已更新')
      return true
    }
    const { data } = await api.post('/cf/accounts', {
      name: form.name.trim(),
      api_token: form.api_token.trim(),
      zone_id: form.zone_id.trim(),
    })
    emit('saved', data?.id)
    emit('update:show', false)
    notifySuccess('保存成功', 'Cloudflare 账号已添加')
    return true
  } finally {
    accountSaving.value = false
  }
}
</script>
