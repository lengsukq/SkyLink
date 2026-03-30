<template>
  <n-modal :show="show" preset="card" title="配置节点" style="width: min(860px, 94vw)" @update:show="emit('update:show', $event)">
    <div class="node-config-modal-body">
      <n-space vertical :size="12">
        <p class="status-hint" style="margin: 0">
          推荐步骤：1）先填写网络名、网络密钥和初始节点；2）点击“保存并重启”；3）回到状态卡查看 Peer 常驻信息与在网信息。
        </p>
        <n-alert type="info" title="节点连接能力">
          当前配置支持同时填写多个初始节点地址（ET_PEERS），系统会按逗号/换行自动识别。
        </n-alert>
        <n-form :model="form" label-placement="left" label-width="120">
          <n-divider style="margin: 0">基础组网</n-divider>
          <n-form-item label="网络名">
            <n-input v-model:value="form.network_name" placeholder="网络标识（ET_NETWORK_NAME）" />
          </n-form-item>
          <n-form-item label="网络密钥">
            <n-input
              v-model:value="form.network_secret"
              type="password"
              show-password-on="click"
              placeholder="网络密钥（ET_NETWORK_SECRET）"
            />
          </n-form-item>
          <n-form-item label="网络方式">
            <n-radio-group :value="networkMode" @update:value="emit('update:networkMode', $event)">
              <n-space>
                <n-radio value="public">公共服务器</n-radio>
                <n-radio value="manual">手动</n-radio>
                <n-radio value="standalone">独立</n-radio>
              </n-space>
            </n-radio-group>
          </n-form-item>
          <n-divider style="margin: 0">节点连接</n-divider>
          <n-form-item v-if="networkMode === 'public'" label="公共服务器">
            <n-input :value="publicServer" placeholder="如 tcp://public.easytier.cn:11010" @update:value="onPublicServerChange" />
            <template #feedback>
              <span class="status-hint">常用格式：`tcp://host:port`，支持 IPv4/域名。</span>
            </template>
          </n-form-item>
          <n-form-item v-else-if="networkMode === 'manual'" label="初始节点（peers）">
            <n-input
              v-model:value="form.peers"
              type="textarea"
              :rows="4"
              placeholder="如 tcp://public.easytier.cn:11010，多个用逗号或换行"
            />
            <template #feedback>
              <span class="status-hint">已识别 {{ peerCount }} 个节点：{{ parsedPeers.slice(0, 3).join('，') || '—' }}<span v-if="peerCount > 3"> ...</span></span>
            </template>
          </n-form-item>
          <n-form-item v-else label="初始节点（peers）">
            <n-input
              type="textarea"
              :rows="2"
              disabled
              placeholder="独立模式下无需填写初始节点。"
            />
          </n-form-item>
          <n-divider style="margin: 0">地址策略</n-divider>
          <n-form-item label="虚拟 IPv4 地址">
            <n-space align="center">
              <n-input
                v-model:value="form.ipv4"
                class="ipv4-input"
                placeholder="如 10.0.111.99"
                :disabled="form.dhcp"
                @update:value="onIpv4Change"
              />
              <span class="cidr-suffix">/24</span>
              <n-checkbox v-model:checked="form.dhcp" @update:checked="onDhcpChange">DHCP</n-checkbox>
            </n-space>
          </n-form-item>
          <n-divider style="margin: 0">高级项（常用）</n-divider>
          <n-form-item label="主机名">
            <n-input v-model:value="form.hostname" placeholder="ET_HOSTNAME（可选）" />
          </n-form-item>
          <n-form-item label="公网发现节点">
            <n-input v-model:value="form.external_node" placeholder="如 tcp://public.easytier.cn:11010（ET_EXTERNAL_NODE）" />
          </n-form-item>
          <n-form-item label="子网代理">
            <n-input
              v-model:value="form.proxy_networks"
              type="textarea"
              :rows="2"
              placeholder="如 10.0.0.0/24，多个子网用逗号或换行（ET_PROXY_NETWORKS）"
            />
          </n-form-item>
          <n-form-item label="RPC 地址">
            <n-input v-model:value="form.rpc_portal" placeholder="如 127.0.0.1:15888（ET_RPC_PORTAL）" />
          </n-form-item>
          <n-form-item label="版本标签">
            <n-input v-model:value="form.image_tag" placeholder="如 latest / v2.2.3（EASYTIER_IMAGE_TAG）" />
          </n-form-item>
          <n-form-item label="启用">
            <n-switch v-model:value="form.enabled" />
          </n-form-item>
        </n-form>
        <n-space justify="end">
          <n-button @click="emit('update:show', false)">取消</n-button>
          <n-button :loading="saving" @click="emit('save')">保存配置</n-button>
          <n-button type="primary" :loading="saving" @click="emit('saveAndRestart')">保存并重启</n-button>
        </n-space>
      </n-space>
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import {
  NModal,
  NSpace,
  NAlert,
  NForm,
  NFormItem,
  NDivider,
  NInput,
  NRadioGroup,
  NRadio,
  NCheckbox,
  NSwitch,
  NButton,
} from 'naive-ui'
import type { EasyTierFormState } from '../../composables/easyTier/types'

const props = defineProps<{
  show: boolean
  form: EasyTierFormState
  networkMode: string
  publicServer: string
  peerCount: number
  parsedPeers: string[]
  saving: boolean
  onDhcpChange: (val: boolean) => void
  onIpv4Change: (val: string) => void
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
  'update:networkMode': [value: string]
  'update:publicServer': [value: string]
  save: []
  saveAndRestart: []
}>()

function onPublicServerChange(value: string) {
  emit('update:publicServer', value)
  props.form.peers = value
}

function onDhcpChange(val: boolean) {
  props.onDhcpChange(val)
}

function onIpv4Change(val: string) {
  props.onIpv4Change(val)
}
</script>

<style scoped>
.node-config-modal-body {
  max-height: 75vh;
  overflow-y: auto;
  padding-right: 4px;
}
.status-hint {
  font-size: 13px;
  color: #64748b;
}
</style>
