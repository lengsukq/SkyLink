<template>
  <div>
    <page-header
      title="文件服务"
      description="WebDAV 将本地目录暴露为 HTTP/WebDAV；Windows、Linux、macOS 均可通过 SkyLink 使用此能力。"
    />
    <web-dev-view embedded />
  </div>
</template>

<script setup>
import PageHeader from '../components/PageHeader.vue'
import WebDevView from './WebDev.vue'
</script>
