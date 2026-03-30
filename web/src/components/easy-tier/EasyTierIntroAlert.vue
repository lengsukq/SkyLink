<template>
  <n-alert type="info" class="page-section mb-0">
    <template #header>推荐步骤</template>
    <p class="et-intro-line">
      ① 在「版本与运行时」选择版本与平台并下载 → ② 在「网络配置」填写网络名、密钥等并启用、保存 → ③ 在状态区选择已下载版本并点击「启动」。
    </p>
    <p class="et-intro-line">
      完整命令行参数与环境变量说明见 EasyTier 官方文档：
      <a href="https://easytier.cn/guide/network/configurations.html" target="_blank" rel="noopener noreferrer">完整配置选项</a>
      （如 <code>ET_NETWORK_NAME</code>、<code>ET_PEERS</code>、<code>ET_RPC_PORTAL</code> 等）。SkyLink 将表单写入数据库并生成 env 供 <code>easytier-core</code> 使用。
    </p>
  </n-alert>
</template>

<script setup lang="ts">
import { NAlert } from 'naive-ui'
</script>

<style scoped>
.et-intro-line {
  margin: 0 0 8px 0;
  font-size: 13px;
  line-height: 1.55;
}
.et-intro-line:last-child {
  margin-bottom: 0;
}
.et-intro-line code {
  font-size: 12px;
  padding: 0 4px;
  border-radius: 3px;
  background: var(--n-color-target, rgba(0, 0, 0, 0.06));
}
</style>
