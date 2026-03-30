<template>
  <n-layout
    class="app-layout min-h-screen bg-white"
  >
    <n-layout-header class="minimal-header" :bordered="false">
      <div class="minimal-header__inner">
        <div class="minimal-header__brand">
          <sky-link-logo size="sm" />
          <div v-if="headerTitle" class="minimal-header__meta">
            <div class="minimal-header__title">{{ headerTitle }}</div>
            <div v-if="headerDescription" class="minimal-header__desc">{{ headerDescription }}</div>
          </div>
        </div>
      </div>
    </n-layout-header>
    <n-layout-content class="app-content app-content--login">
      <div class="app-content__inner" :class="isDrivePortalRoute ? 'minimal-content__inner--full' : 'minimal-content__inner'">
        <router-view />
      </div>
    </n-layout-content>
  </n-layout>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { NLayout, NLayoutContent, NLayoutHeader } from 'naive-ui'
import SkyLinkLogo from '../components/SkyLinkLogo.vue'

const route = useRoute()

const headerMetaByRouteName: Record<string, { title: string; description?: string }> = {
  DrivePortal: {
    title: '个人网盘',
    description: '文件浏览、上传下载、类型分类筛选（支持递归搜索）。',
  },
}

const headerTitle = computed(() => {
  const key = String(route.name || '')
  return headerMetaByRouteName[key]?.title || ''
})

const headerDescription = computed(() => {
  const key = String(route.name || '')
  return headerMetaByRouteName[key]?.description || ''
})

const isDrivePortalRoute = computed(() => String(route.name || '') === 'DrivePortal')
</script>

<style scoped>
.app-layout {
  min-height: 100vh;
}

.minimal-header {
  height: 56px;
  display: flex;
  align-items: center;
  border-bottom: 1px solid #f1f5f9;
  background: #ffffff;
}

.minimal-header__inner {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 24px;
}

.minimal-header__brand {
  display: flex;
  align-items: center;
  gap: 14px;
}

.minimal-header__meta {
  display: flex;
  flex-direction: column;
  line-height: 1.25;
}

.minimal-header__title {
  font-size: 14px;
  font-weight: 600;
  color: #0f172a;
}

.minimal-header__desc {
  font-size: 12px;
  color: #64748b;
}

.app-content {
  padding: 20px 24px 28px;
}

.app-content__inner {
  width: 100%;
}

.minimal-content__inner {
  max-width: 1200px;
  margin: 0 auto;
}

.minimal-content__inner--full {
  width: 100%;
  max-width: none;
  margin: 0;
}

@media (max-width: 640px) {
  .minimal-header__inner {
    padding: 0 16px;
  }

  .minimal-header__desc {
    display: none;
  }

  .app-content {
    padding: 16px 16px 24px;
  }
}
</style>
