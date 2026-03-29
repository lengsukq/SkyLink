<template>
  <component
    :is="to ? RouterLink : 'div'"
    v-bind="to ? { to } : {}"
    :data-size="size"
    class="skylink-logo inline-flex shrink-0 items-center no-underline"
    :class="[sizeClass, { 'cursor-default': !to, 'cursor-pointer hover:opacity-90': to }]"
  >
    <span class="skylink-logo__mark text-sky-600" aria-hidden="true">
      <svg
        :width="iconPx"
        :height="iconPx"
        viewBox="0 0 24 24"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
      >
        <path
          d="M10 13a5 5 0 0 1 0-7l1-1a5 5 0 0 1 7 7l-1 1M14 11a5 5 0 0 1 0 7l-1 1a5 5 0 0 1-7-7l1-1"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
      </svg>
    </span>
    <span class="skylink-logo__wordmark tracking-tight" :class="wordmarkClass">
      <span class="text-slate-800">Sky</span><span class="skylink-logo__accent">Link</span>
    </span>
  </component>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { RouteLocationRaw } from 'vue-router'
import { RouterLink } from 'vue-router'

const props = withDefaults(
  defineProps<{
    /** 顶栏紧凑 / 登录页大标题 / 中等（子页品牌条） */
    size?: 'sm' | 'md' | 'lg'
    /** 可点击回到该路由（如管理首页） */
    to?: RouteLocationRaw
  }>(),
  { size: 'sm' },
)

const iconPx = computed(() => ({ sm: 22, md: 28, lg: 40 }[props.size]))

const sizeClass = computed(() => {
  switch (props.size) {
    case 'lg':
      return 'gap-3 text-[2.25rem] sm:text-[2.75rem] leading-none'
    case 'md':
      return 'gap-2.5 text-xl leading-none'
    default:
      return 'gap-2 text-[1.125rem] leading-none'
  }
})

const wordmarkClass = computed(() => {
  switch (props.size) {
    case 'lg':
      return 'font-extrabold'
    case 'md':
      return 'font-semibold'
    default:
      return 'font-semibold'
  }
})
</script>

<style scoped>
.skylink-logo__accent {
  background: linear-gradient(120deg, #0ea5e9 0%, #6366f1 45%, #8b5cf6 100%);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}

.skylink-logo[data-size='lg'] .skylink-logo__accent {
  background: linear-gradient(120deg, #38bdf8 0%, #818cf8 45%, #a855f7 90%);
  -webkit-background-clip: text;
  background-clip: text;
}
</style>
