import { type Ref } from 'vue'
import api from '../../api/client'
import { notifyError, notifySuccess } from '../../ui/notify'
import { getApiErrorMessage } from '../../utils/apiError'
import type { EasyTierFormState } from './types'

type UseEasyTierConfigFormOptions = {
  form: EasyTierFormState
  networkMode: Ref<string>
  parsedPeers: Ref<string[]>
  publicServer: Ref<string>
  saving: Ref<boolean>
  profilePath: (path: string) => string
  loadConfig: () => Promise<void>
  loadStatus: () => Promise<void>
  loadDaemonStatus: () => Promise<void>
}

export function useEasyTierConfigForm(options: UseEasyTierConfigFormOptions) {
  function getHint(error: unknown): string {
    if (error == null || typeof error !== 'object') return ''
    const response = (error as { response?: { data?: unknown } }).response
    const data = response?.data
    if (data == null || typeof data !== 'object') return ''
    const hint = (data as { hint?: unknown }).hint
    return typeof hint === 'string' ? hint : ''
  }

  function validateIpv4ForSave(errors: string[]) {
    const ipv4 = options.form.ipv4.trim()
    const ipv4Pattern = /^(25[0-5]|2[0-4]\d|1?\d?\d)(\.(25[0-5]|2[0-4]\d|1?\d?\d)){3}$/
    if (!options.form.dhcp && !ipv4) {
      errors.push('未启用 DHCP 时必须填写本机 IPv4。')
      return
    }
    if (ipv4 && !ipv4Pattern.test(ipv4)) {
      errors.push('本机 IPv4 格式不正确，请使用形如 10.144.144.1 的地址。')
    }
  }

  function normalizePeersByNetworkMode() {
    if (options.networkMode.value === 'standalone') {
      options.form.peers = ''
    } else if (options.networkMode.value === 'public') {
      options.form.peers = (options.publicServer.value || '').trim()
    }
  }

  function validateForSave(errors: string[]) {
    if (options.form.enabled) {
      if (!options.form.network_name.trim()) {
        errors.push('请填写网络名（ET_NETWORK_NAME，对应 EasyTier network_identity.name）。')
      }
      if (!options.form.network_secret.trim()) {
        errors.push('请填写网络密钥（ET_NETWORK_SECRET，对应 EasyTier network_identity.secret）。')
      }
      if (options.networkMode.value !== 'standalone' && options.parsedPeers.value.length === 0) {
        errors.push('请至少填写一个初始节点（ET_PEERS，对应 peers[].uri）。')
      }
    }
    validateIpv4ForSave(errors)
    const cidrPattern = /^(25[0-5]|2[0-4]\d|1?\d?\d)(\.(25[0-5]|2[0-4]\d|1?\d?\d)){3}\/([0-9]|[12][0-9]|3[0-2])$/
    if (options.form.proxy_networks.trim()) {
      const items = options.form.proxy_networks
        .split(/[\n,]+/)
        .map((s) => s.trim())
        .filter(Boolean)
      const invalid = items.filter((item) => !cidrPattern.test(item))
      if (invalid.length) {
        errors.push(`子网代理（proxy-networks）格式不正确：${invalid.join('，')}。请使用形如 10.0.0.0/24 的 CIDR。`)
      }
    }
  }

  function validateForSaveAndRestart(errors: string[]) {
    if (options.form.enabled) {
      if (!options.form.network_name.trim()) errors.push('请填写网络名。')
      if (!options.form.network_secret.trim()) errors.push('请填写网络密钥。')
      if (options.networkMode.value !== 'standalone' && options.parsedPeers.value.length === 0) {
        errors.push('请至少填写一个初始节点。')
      }
    }
    validateIpv4ForSave(errors)
    const cidrPattern = /^(25[0-5]|2[0-4]\d|1?\d?\d)(\.(25[0-5]|2[0-4]\d|1?\d?\d)){3}\/([0-9]|[12][0-9]|3[0-2])$/
    if (options.form.proxy_networks.trim()) {
      const items = options.form.proxy_networks.split(/[\n,]+/).map((s) => s.trim()).filter(Boolean)
      const invalid = items.filter((item) => !cidrPattern.test(item))
      if (invalid.length) errors.push(`子网代理格式不正确：${invalid.join('，')}`)
    }
  }

  async function save() {
    const errors: string[] = []
    validateForSave(errors)
    if (errors.length) {
      notifyError('配置不完整或格式错误', errors.join('；'))
      return false
    }
    normalizePeersByNetworkMode()
    options.saving.value = true
    try {
      await api.put('/easytier/config', options.form)
      notifySuccess('已保存', '若需使网络/高级配置生效，请点击上方状态卡片中的「重启」。')
      await options.loadConfig()
      return true
    } catch (e: unknown) {
      notifyError('保存失败', getApiErrorMessage(e, '保存失败'))
      return false
    } finally {
      options.saving.value = false
    }
  }

  async function saveAndRestart() {
    const errors: string[] = []
    validateForSaveAndRestart(errors)
    if (errors.length) {
      notifyError('配置不完整或格式错误', errors.join('；'))
      return false
    }
    normalizePeersByNetworkMode()
    options.saving.value = true
    try {
      await api.put('/easytier/config', options.form)
      await options.loadConfig()
      try {
        const { data } = await api.post(options.profilePath('/daemon/restart'))
        notifySuccess('已保存并重启', data?.message || '配置已保存，EasyTier 守护进程已重启。')
        await options.loadDaemonStatus()
        await options.loadStatus()
        return true
      } catch (e: unknown) {
        const msg = getApiErrorMessage(e, '重启失败')
        const hint = getHint(e)
        notifyError('配置已保存，但重启失败', hint ? `${msg} ${hint}` : msg)
        await options.loadDaemonStatus()
        return false
      }
    } catch (e: unknown) {
      notifyError('保存失败', getApiErrorMessage(e, '保存失败'))
      return false
    } finally {
      options.saving.value = false
    }
  }

  async function saveByModal() {
    const ok = await save()
    if (ok) {
      await options.loadStatus()
    }
    return ok
  }

  async function saveAndRestartByModal() {
    const ok = await saveAndRestart()
    if (ok) {
      await options.loadStatus()
    }
    return ok
  }

  return {
    save,
    saveAndRestart,
    saveByModal,
    saveAndRestartByModal,
  }
}
