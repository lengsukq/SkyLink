<template>
  <div class="empty-state">
    <n-empty :description="title">
      <template #extra>
        <p v-if="description" class="empty-state__description">
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

<style scoped>
.empty-state {
  padding: 24px 0;
}

.empty-state__description {
  margin: 0 0 8px;
  font-size: 13px;
  color: #666;
}
</style>

