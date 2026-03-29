<template>
  <div class="login-page skylink-auth-bg">
    <div class="login-page__content">
      <div class="login-page__brand">
        <sky-link-logo size="lg" />
        <p class="login-page__brand-subtitle">
          智能 DNS 映射与边缘控制台，统一管理你的域名与流量入口。
        </p>
      </div>
      <div class="login-page__card-wrap">
        <div class="login-page__card skylink-glass-card-shell">
          <n-card class="login-page__card-inner skylink-glass-card-inner" title="SkyLink 控制台登录">
            <n-form
              :model="form"
              label-placement="left"
              label-width="80"
              @submit.prevent="onLogin"
            >
              <n-form-item label="密码" required>
                <n-input
                  v-model:value="form.password"
                  type="password"
                  show-password-on="click"
                  placeholder="启动日志里打印的密码"
                  :disabled="loading"
                  @keyup.enter="onLogin"
                />
              </n-form-item>
            </n-form>
            <n-space justify="end">
              <n-button
                type="primary"
                :loading="loading"
                :disabled="!form.password.trim()"
                @click="onLogin"
              >
                使用密码登录
              </n-button>
            </n-space>
            <p class="login-page__hint">
              首次启动会在服务日志里打印一次随机密码，登录后可在「设置」中修改。
              如遗忘密码，可重启服务并查看最新启动日志。
            </p>
          </n-card>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NForm, NFormItem, NInput, NButton, NSpace } from 'naive-ui'
import api from '../api/client'
import { notifySuccess } from '../ui/notify'
import { STORAGE_KEYS } from '../constants/storage'
import { ROUTE_PATHS } from '../constants/routes'
import SkyLinkLogo from '../components/SkyLinkLogo.vue'

const router = useRouter()
const loading = ref(false)
const form = reactive({ password: '' })

async function onLogin() {
  if (!form.password.trim()) return
  loading.value = true
  try {
    await api.post('/auth/login', { password: form.password })
    localStorage.setItem(STORAGE_KEYS.skylinkToken, form.password.trim())
    notifySuccess('登录成功', '欢迎回来')
    await router.replace(ROUTE_PATHS.dashboard)
  } finally {
    loading.value = false
  }
}
</script>
