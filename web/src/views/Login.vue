<template>
  <div class="login-page">
    <div class="login-page__card-wrap">
      <n-card title="登录 SkyLink 管理端">
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
          首次启动会在服务日志里打印一次随机密码。登录后可在“设置”中修改。
          如遗忘密码，可重启服务并查看最新启动日志。
        </p>
      </n-card>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NForm, NFormItem, NInput, NButton, NSpace } from 'naive-ui'
import api from '../api/client'
import { notifySuccess } from '../ui/notify'

const router = useRouter()
const loading = ref(false)
const form = reactive({ password: '' })

async function onLogin() {
  if (!form.password.trim()) return
  loading.value = true
  try {
    await api.post('/auth/login', { password: form.password })
    localStorage.setItem('skylink_token', form.password.trim())
    notifySuccess('登录成功', '欢迎回来')
    await router.replace('/dashboard')
  } finally {
    loading.value = false
  }
}
</script>

<style>
.login-page {
  min-height: calc(100vh - 56px);
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 64px;
  background: linear-gradient(180deg, #f5f7fb 0%, #ffffff 40%);
}

.login-page__card-wrap {
  width: 100%;
  max-width: 420px;
}

.login-page__hint {
  color: #666;
  font-size: 12px;
  margin-top: 12px;
  line-height: 1.6;
}
</style>

