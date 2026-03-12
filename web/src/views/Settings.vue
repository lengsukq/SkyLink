<template>
  <div>
    <page-header
      title="设置"
      description="管理控制台登录密码，以及一键映射时的默认 Cloudflare / FRP 配置。"
    />
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
        <n-button :disabled="loading || defaultsSaving" @click="logout">退出登录</n-button>
      </n-space>
      <p class="settings-hint">
        修改成功后会自动更新本地保存的登录密码，下次访问将直接使用新密码登录。
      </p>
    </n-card>

    <n-card title="FRP / Cloudflare 默认配置" style="max-width: 520px; margin-top: 16px">
      <n-form :model="defaultsForm" label-placement="left" label-width="140">
        <n-form-item label="FRP 固定域名（CNAME 目标）">
          <n-input v-model:value="defaultsForm.frp_cname_target" placeholder="例如：xxx.sakurafrp.com（可留空）" />
        </n-form-item>
        <n-form-item label="CNAME 默认 proxied">
          <n-switch v-model:value="defaultsForm.cf_cname_proxied" />
        </n-form-item>
      </n-form>
      <n-space>
        <n-button type="primary" :loading="defaultsSaving" @click="saveDefaults">保存</n-button>
      </n-space>
      <p class="settings-hint">
        一键映射时如不填写 CNAME 目标，将自动使用这里的默认值。
      </p>
    </n-card>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NForm, NFormItem, NInput, NButton, NSpace, NSwitch } from 'naive-ui'
import api from '../api/client'
import { notifySuccess } from '../ui/notify'
import PageHeader from '../components/PageHeader.vue'

const router = useRouter()
const loading = ref(false)
const pwForm = reactive({ old_password: '', new_password: '' })

const defaultsSaving = ref(false)
const defaultsForm = reactive({ frp_cname_target: '', cf_cname_proxied: true })

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
    notifySuccess('修改成功', '管理密码已更新')
  } finally {
    loading.value = false
  }
}

function logout() {
  localStorage.removeItem('skylink_token')
  router.replace('/login')
}

async function loadDefaults() {
  const { data } = await api.get('/settings')
  defaultsForm.frp_cname_target = (data?.frp_cname_target || '').toString()
  defaultsForm.cf_cname_proxied = !!data?.cf_cname_proxied
}

async function saveDefaults() {
  defaultsSaving.value = true
  try {
    await api.put('/settings', {
      frp_cname_target: (defaultsForm.frp_cname_target || '').trim(),
      cf_cname_proxied: !!defaultsForm.cf_cname_proxied,
    })
    notifySuccess('保存成功', '默认配置已更新')
  } finally {
    defaultsSaving.value = false
  }
}

onMounted(loadDefaults)
</script>

<style>
.settings-hint {
  color: #666;
  font-size: 12px;
  margin-top: 8px;
  line-height: 1.6;
}
</style>
