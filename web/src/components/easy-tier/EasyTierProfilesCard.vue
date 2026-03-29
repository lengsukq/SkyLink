<template>
  <n-card title="配置实例" class="page-section page-card">
    <n-space align="center" wrap>
      <n-select
        v-model:value="activeProfileId"
        :options="profileOptions"
        placeholder="选择配置实例"
        class="min-w-[240px]"
        @update:value="onProfileChange"
      />
      <n-input v-model:value="newProfileName" placeholder="新实例名称" class="w-[180px]" />
      <n-button size="small" @click="emit('create')">新增实例</n-button>
      <n-button size="small" :disabled="!canDelete" @click="emit('delete')">删除当前实例</n-button>
    </n-space>
  </n-card>
</template>

<script setup lang="ts">
import { NCard, NSpace, NSelect, NInput, NButton } from 'naive-ui'

const activeProfileId = defineModel<string>('activeProfileId', { required: true })
const newProfileName = defineModel<string>('newProfileName', { required: true })

defineProps<{
  profileOptions: { label: string; value: string }[]
  canDelete: boolean
}>()

const emit = defineEmits<{
  profileChange: [id: string]
  create: []
  delete: []
}>()

function onProfileChange(id: string) {
  emit('profileChange', id)
}
</script>
