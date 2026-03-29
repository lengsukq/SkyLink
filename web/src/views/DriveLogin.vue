<template>
  <div class="login-page skylink-auth-bg">
    <div class="login-page__content">
      <div class="login-page__brand">
        <sky-link-logo size="lg" />
        <p class="login-page__brand-subtitle">
          使用管理员为你开通的账号登录，在浏览器中访问与管理个人文件空间。
        </p>
      </div>
      <div class="login-page__card-wrap">
        <div class="login-page__card skylink-glass-card-shell">
          <n-card class="login-page__card-inner skylink-glass-card-inner" title="个人网盘登录">
            <n-form :model="form" label-placement="top" @submit.prevent="login">
              <n-form-item label="用户名">
                <n-input
                  v-model:value="form.username"
                  placeholder="请输入用户名"
                  :disabled="loading"
                  @keyup.enter="focusPassword"
                />
              </n-form-item>
              <n-form-item label="密码">
                <n-input
                  ref="passwordInputRef"
                  v-model:value="form.password"
                  type="password"
                  placeholder="请输入密码"
                  show-password-on="click"
                  :disabled="loading"
                  @keyup.enter="login"
                />
              </n-form-item>
            </n-form>
            <n-button type="primary" block :loading="loading" @click="login">登录</n-button>
            <p class="login-page__hint">
              网盘账号由管理员在控制台「设置」中创建与分配；若无法登录，请联系管理员确认账号或重置密码。
            </p>
            <p class="login-page__footer-link">
              <router-link :to="ROUTE_PATHS.login">管理员登录</router-link>
            </p>
          </n-card>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import type { InputInst } from 'naive-ui'
import { NButton, NCard, NForm, NFormItem, NInput } from 'naive-ui'
import api from '../api/client'
import { STORAGE_KEYS } from '../constants/storage'
import { ROUTE_PATHS } from '../constants/routes'
import SkyLinkLogo from '../components/SkyLinkLogo.vue'
import { notifyError, notifySuccess } from '../ui/notify'

const router = useRouter()
const loading = ref(false)
const form = ref({ username: '', password: '' })
const passwordInputRef = ref<InputInst | null>(null)

function focusPassword() {
  passwordInputRef.value?.focus()
}

async function login() {
  const username = form.value.username.trim()
  const password = form.value.password
  if (!username || !password.trim()) {
    notifyError('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    const { data } = await api.post('/drive/auth/login', { username, password }, { silentError: true })
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
