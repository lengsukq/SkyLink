<template>
  <n-config-provider :theme-overrides="themeOverrides">
    <n-notification-provider>
      <notifier-bridge />
      <n-message-provider>
        <router-view />
      </n-message-provider>
    </n-notification-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import {
  NConfigProvider,
  NMessageProvider,
  NNotificationProvider,
} from 'naive-ui'
import NotifierBridge from './components/NotifierBridge.vue'

const themeOverrides = {
  common: {
    fontFamily: 'system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
    borderRadius: '16px',
    borderRadiusSmall: '12px',
    boxShadow1: '0 1px 2px 0 rgba(15, 23, 42, 0.05)',
    boxShadow2: '0 8px 24px -6px rgba(15, 23, 42, 0.08)',
    boxShadow3: '0 20px 48px -12px rgba(15, 23, 42, 0.12)',
    primaryColor: '#0284c7',
    primaryColorHover: '#0ea5e9',
    primaryColorPressed: '#0369a1',
    primaryColorSuppl: '#0ea5e9',
    infoColor: '#0284c7',
    infoColorHover: '#0ea5e9',
    infoColorPressed: '#0369a1',
  },
  Card: {
    borderRadius: '16px',
  },
  Button: {
    borderRadiusTiny: '10px',
    borderRadiusSmall: '12px',
    borderRadiusMedium: '14px',
    borderRadiusLarge: '16px',
  },
  Input: {
    borderRadius: '14px',
  },
  Select: {
    peers: {
      InternalSelection: {
        borderRadius: '14px',
      },
    },
  },
  DataTable: {
    borderRadius: '14px',
  },
  Dialog: {
    borderRadius: '18px',
  },
}
</script>

<style>
* {
  box-sizing: border-box;
}

body {
  margin: 0;
}

#app {
  min-height: 100vh;
}
</style>
