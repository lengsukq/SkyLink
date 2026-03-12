<template>
  <div class="page-header">
    <div class="page-header__main">
      <div>
        <div class="page-header__title-row">
          <n-h1 class="page-header__title">
            {{ title }}
          </n-h1>
        </div>
        <p v-if="description" class="page-header__description">
          {{ description }}
        </p>
      </div>
      <div v-if="$slots.actions" class="page-header__actions">
        <slot name="actions" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { NH1 } from 'naive-ui'

defineProps({
  title: {
    type: String,
    required: true,
  },
  description: {
    type: String,
    default: '',
  },
})
</script>

<style scoped>
.page-header {
  margin-bottom: 16px;
}

.page-header__main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.page-header__title {
  margin: 0;
}

.page-header__description {
  margin: 4px 0 0;
  font-size: 13px;
  color: #666;
}

.page-header__actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>

