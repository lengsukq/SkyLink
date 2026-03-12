<template>
  <div style="max-width: 420px; margin: 48px auto;">
    <n-card title="登录 SkyLink 管理端">
      <n-form :model="form" label-placement="left" label-width="80">
        <n-form-item label="密码" required>
          <n-input v-model:value="form.password" type="password" show-password-on="click" placeholder="启动日志里打印的密码" />
        </n-form-item>
      </n-form>
      <n-space justify="end">
        <n-button type="primary" :loading="loading" @click="onLogin">登录</n-button>
      </n-space>
      <p style="color:#666;font-size:12px;margin-top:12px">
        首次启动会在服务日志里打印一次随机密码。登录后可在“设置”中修改。
      </p>
    </n-card>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NForm, NFormItem, NInput, NButton, NSpace } from 'naive-ui'
import api from '../api/client'

const router = useRouter()
const loading = ref(false)
const form = reactive({ password: '' })

async function onLogin() {
  if (!form.password.trim()) return
  loading.value = true
  try {
    await api.post('/auth/login', { password: form.password })
    localStorage.setItem('skylink_token', form.password.trim())
    await router.replace('/dashboard')
  } finally {
    loading.value = false
  }
}
</script>

