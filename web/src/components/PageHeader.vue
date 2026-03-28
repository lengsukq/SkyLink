<template>
  <div class="mb-5">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-baseline gap-2">
          <n-h1 class="!m-0 text-2xl font-semibold tracking-tight text-slate-900 sm:text-[1.65rem]">
            {{ title }}
          </n-h1>
        </div>
        <p v-if="description" class="mt-1.5 text-[13px] leading-relaxed text-slate-500">
          {{ description }}
        </p>
      </div>
      <div v-if="$slots.actions" class="flex shrink-0 flex-wrap items-center gap-2">
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
