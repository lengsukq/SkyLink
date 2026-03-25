<template>
  <div>
    <page-header
      title="设置"
      description="管理控制台登录密码，以及一键映射时的默认 Cloudflare / FRP 配置。"
    />
    <n-grid cols="1 s:1 m:2 l:2 xl:3" x-gap="16" y-gap="16" class="page-section">
      <n-gi>
        <n-card title="管理密码" class="page-card">
          <n-form :model="pwForm" label-placement="left" label-width="100">
            <n-form-item label="当前密码" required>
              <n-input v-model:value="pwForm.old_password" type="password" show-password-on="click" placeholder="当前登录密码" />
            </n-form-item>
            <n-form-item label="新密码" required>
              <n-input v-model:value="pwForm.new_password" type="password" show-password-on="click" placeholder="设置一个新密码" />
            </n-form-item>
            <n-form-item label="确认新密码" required>
              <n-input
                v-model:value="pwForm.confirm_password"
                type="password"
                show-password-on="click"
                placeholder="再次输入新密码"
              />
            </n-form-item>
          </n-form>
          <n-space>
            <n-button type="primary" :loading="loading" @click="changePassword">修改密码</n-button>
            <n-button :disabled="loading || defaultsSaving" @click="logout">退出登录</n-button>
          </n-space>
          <p class="settings-hint">
            修改成功后会自动更新本地保存的登录密码。
          </p>
        </n-card>
      </n-gi>

      <n-gi>
        <n-card title="FRP / Cloudflare 默认配置" class="page-card">
          <n-form :model="defaultsForm" label-placement="left" label-width="140">
            <n-form-item label="FRP 固定域名（CNAME 目标）">
              <n-input v-model:value="defaultsForm.frp_cname_target" placeholder="例如：xxx.sakurafrp.com" />
            </n-form-item>
            <n-form-item label="CNAME 默认 proxied">
              <n-switch v-model:value="defaultsForm.cf_cname_proxied" />
            </n-form-item>
          </n-form>
          <n-space>
            <n-button type="primary" :loading="defaultsSaving" @click="saveDefaults">保存</n-button>
          </n-space>
          <p class="settings-hint">
            一键映射会在 Cloudflare 中创建指向此 FRP 固定域名的 CNAME 记录。
          </p>
        </n-card>
      </n-gi>

      <n-gi>
        <n-card title="EasyTier" class="page-card">
          <n-form label-placement="left" label-width="180">
            <n-form-item label="SkyLink 启动时自动启动 EasyTier">
              <n-switch
                :value="easytierAutostart"
                :loading="easytierAutostartSaving"
                @update:value="onToggleEasyTierAutostart"
              />
            </n-form-item>
          </n-form>
          <p class="settings-hint">
            需已在 EasyTier 页面完成网络名、网络密钥和初始节点（peers）配置并开启「启用」。该开关仅控制 SkyLink 进程启动时是否自动拉起 EasyTier 守护进程，页面上的「启动 / 重启」按钮行为不受影响。
          </p>
        </n-card>
      </n-gi>
    </n-grid>

  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NForm, NFormItem, NInput, NButton, NSpace, NSwitch, NGrid, NGi } from 'naive-ui'
import api from '../api/client'
import { notifySuccess, notifyError } from '../ui/notify'
import PageHeader from '../components/PageHeader.vue'
import { STORAGE_KEYS } from '../constants/storage'
import { ROUTE_PATHS } from '../constants/routes'

const router = useRouter()
const loading = ref(false)
const pwForm = reactive({ old_password: '', new_password: '', confirm_password: '' })

const defaultsSaving = ref(false)
const defaultsForm = reactive({ frp_cname_target: '', cf_cname_proxied: true })

const easytierAutostart = ref(false)
const easytierAutostartSaving = ref(false)

async function changePassword() {
  if (!pwForm.old_password.trim() || !pwForm.new_password.trim() || !pwForm.confirm_password.trim()) {
    notifyError('信息不完整', '请填写当前密码、新密码和确认新密码。')
    return
  }
  if (pwForm.new_password.trim() !== pwForm.confirm_password.trim()) {
    notifyError('两次密码不一致', '请确保两次输入的新密码完全一致。')
    return
  }
  loading.value = true
  try {
    await api.post('/auth/password', {
      old_password: pwForm.old_password.trim(),
      new_password: pwForm.new_password.trim(),
    })
    localStorage.setItem(STORAGE_KEYS.skylinkToken, pwForm.new_password.trim())
    pwForm.old_password = ''
    pwForm.new_password = ''
    pwForm.confirm_password = ''
    notifySuccess('修改成功', '管理密码已更新')
  } finally {
    loading.value = false
  }
}

function logout() {
  localStorage.removeItem(STORAGE_KEYS.skylinkToken)
  router.replace(ROUTE_PATHS.login)
}

async function loadDefaults() {
  const { data } = await api.get('/settings')
  defaultsForm.frp_cname_target = (data?.frp_cname_target || '').toString()
  defaultsForm.cf_cname_proxied = !!data?.cf_cname_proxied
}

async function loadEasyTierSettings() {
  const { data } = await api.get('/easytier/settings')
  easytierAutostart.value = !!data?.autostart_on_startup
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

async function onToggleEasyTierAutostart(val) {
  const previous = easytierAutostart.value
  easytierAutostart.value = val
  easytierAutostartSaving.value = true
  try {
    await api.put('/easytier/settings', {
      autostart_on_startup: !!val,
    })
    notifySuccess('已更新', 'EasyTier 自动启动开关已更新')
  } catch (e) {
    // 若后端校验失败或保存异常，回滚到之前的值；具体错误提示由全局拦截器处理
    easytierAutostart.value = previous
  } finally {
    easytierAutostartSaving.value = false
  }
}

onMounted(async () => {
  await Promise.all([loadDefaults(), loadEasyTierSettings()])
})
</script>

<style>
.settings-hint {
  color: #666;
  font-size: 11px;
  margin-top: 4px;
  line-height: 1.5;
}

.settings-hint--inline {
  margin: 0 0 8px 0;
}
</style>
