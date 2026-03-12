<template>
  <n-config-provider :theme-overrides="themeOverrides">
    <n-message-provider>
      <n-layout style="min-height: 100vh">
        <n-layout-header bordered style="height: 56px; padding: 0 20px; display: flex; align-items: center; gap: 16px">
          <span style="font-weight: 600; font-size: 1.1rem">SkyLink</span>
          <n-space>
            <n-button quaternary tag="a" href="#/dashboard">仪表盘</n-button>
            <n-button quaternary tag="a" href="#/mappings">映射</n-button>
            <n-button quaternary tag="a" href="#/cloudflare">Cloudflare</n-button>
            <n-button quaternary tag="a" href="#/ddns">DDNS</n-button>
            <n-button quaternary tag="a" href="#/settings">设置</n-button>
          </n-space>
        </n-layout-header>
        <n-layout-content content-style="padding: 20px">
          <router-view />
        </n-layout-content>
      </n-layout>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup>
import { NConfigProvider, NLayout, NLayoutHeader, NLayoutContent, NSpace, NButton, NMessageProvider } from 'naive-ui'

const themeOverrides = {
  common: {
    borderRadius: '6px',
  },
}
</script>

<style>
* { box-sizing: border-box; }
body { margin: 0; font-family: system-ui, sans-serif; }
#app { min-height: 100vh; }
</style>
