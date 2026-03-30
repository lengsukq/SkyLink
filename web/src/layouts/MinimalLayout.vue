<template>
  <n-layout
    class="app-layout min-h-screen bg-white"
  >
    <n-layout-content class="app-content app-content--login !p-0">
      <div class="app-content__inner">
        <router-view />
      </div>
    </n-layout-content>
  </n-layout>
</template>

<script setup lang="ts">
import { NLayout, NLayoutContent } from 'naive-ui'
</script>

<style scoped>
.app-layout {
  min-height: 100vh;
}

.app-content__inner {
  width: 100%;
  max-width: none;
  margin: 0;
}
</style>
