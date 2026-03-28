<template>
  <div class="py-8">
    <n-empty :description="title">
      <template #extra>
        <p v-if="description" class="mb-2 text-[13px] text-slate-500">
          {{ description }}
        </p>
        <n-button
          v-if="primaryText"
          type="primary"
          size="small"
          @click="onPrimary"
        >
          {{ primaryText }}
        </n-button>
      </template>
    </n-empty>
  </div>
</template>

<script setup>
import { NEmpty, NButton } from 'naive-ui'

const props = defineProps({
  title: {
    type: String,
    required: true,
  },
  description: {
    type: String,
    default: '',
  },
  primaryText: {
    type: String,
    default: '',
  },
})

const emit = defineEmits(['primary'])

function onPrimary() {
  emit('primary')
}
</script>
