<template>
  <div class="login">
    <n-card class="login-card">
      <n-space vertical size="large">
        <div class="title">个人网盘登录</div>
        <n-form :model="form" label-placement="top">
          <n-form-item label="用户名">
            <n-input v-model:value="form.username" placeholder="请输入用户名" />
          </n-form-item>
          <n-form-item label="密码">
            <n-input v-model:value="form.password" type="password" placeholder="请输入密码" show-password-on="click" />
          </n-form-item>
        </n-form>
        <n-button type="primary" block :loading="loading" @click="login">登录</n-button>
      </n-space>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NCard, NForm, NFormItem, NInput, NSpace } from 'naive-ui'
import api from '../api/client'
import { STORAGE_KEYS } from '../constants/storage'
import { ROUTE_PATHS } from '../constants/routes'
import { notifyError, notifySuccess } from '../ui/notify'

const router = useRouter()
const loading = ref(false)
const form = ref({ username: '', password: '' })

async function login() {
  const username = form.value.username.trim()
  const password = form.value.password
  if (!username || !password.trim()) {
    notifyError('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    const { data } = await api.post('/drive/auth/login', { username, password }, { silentError: true } as any)
    const token = String(data?.token || '')
    if (!token) {
      notifyError('登录失败', '未获取到 token')
      return
    }
    localStorage.setItem(STORAGE_KEYS.driveUserToken, token)
    notifySuccess('登录成功', '欢迎使用个人网盘')
    router.push(ROUTE_PATHS.drivePortal)
  } catch (e: any) {
    notifyError('登录失败', e?.response?.data?.error || e?.message || String(e))
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}
.login-card {
  width: 420px;
}
.title {
  font-size: 18px;
  font-weight: 700;
}
</style>

