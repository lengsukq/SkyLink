<template>
  <div>
    <n-h1>设置</n-h1>
    <n-card title="管理密码" style="max-width: 520px; margin-top: 16px">
      <n-form :model="pwForm" label-placement="left" label-width="100">
        <n-form-item label="当前密码" required>
          <n-input v-model:value="pwForm.old_password" type="password" show-password-on="click" placeholder="当前登录密码" />
        </n-form-item>
        <n-form-item label="新密码" required>
          <n-input v-model:value="pwForm.new_password" type="password" show-password-on="click" placeholder="设置一个新密码" />
        </n-form-item>
      </n-form>
      <n-space>
        <n-button type="primary" :loading="loading" @click="changePassword">修改密码</n-button>
        <n-button @click="logout">退出登录</n-button>
      </n-space>
      <p style="color: #666; font-size: 12px; margin-top: 8px">
        修改成功后会自动更新本地保存的登录密码。
      </p>
    </n-card>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NForm, NFormItem, NInput, NButton, NH1, NSpace } from 'naive-ui'
import api from '../api/client'

const router = useRouter()
const loading = ref(false)
const pwForm = reactive({ old_password: '', new_password: '' })

async function changePassword() {
  if (!pwForm.old_password.trim() || !pwForm.new_password.trim()) return
  loading.value = true
  try {
    await api.post('/auth/password', {
      old_password: pwForm.old_password.trim(),
      new_password: pwForm.new_password.trim(),
    })
    localStorage.setItem('skylink_token', pwForm.new_password.trim())
    pwForm.old_password = ''
    pwForm.new_password = ''
  } finally {
    loading.value = false
  }
}

function logout() {
  localStorage.removeItem('skylink_token')
  router.replace('/login')
}
</script>
