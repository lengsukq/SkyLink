<template>
  <n-card v-if="queue.length" size="small" class="uploader mt-1.5">
    <n-space vertical>
      <div class="uploader-title">上传队列</div>
      <n-space v-for="item in queue" :key="item.id" justify="space-between" align="center">
        <div class="uploader-name">{{ item.name }}</div>
        <div class="uploader-right">
          <n-progress
            type="line"
            :percentage="item.percent"
            :processing="item.status === 'uploading'"
            indicator-placement="inside"
          />
        </div>
      </n-space>
    </n-space>
  </n-card>
</template>

<script setup lang="ts">
import { NCard, NSpace, NProgress } from 'naive-ui'
import type { DriveUploadQueueItem } from '../../composables/drive/useDriveUploadQueue'

defineProps<{
  queue: DriveUploadQueueItem[]
}>()
</script>

<style scoped>
.uploader-title {
  font-weight: 600;
  font-size: 13px;
}
.uploader-name {
  max-width: 360px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.uploader-right {
  width: 260px;
}
</style>
