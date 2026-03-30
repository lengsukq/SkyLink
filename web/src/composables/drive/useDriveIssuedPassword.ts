import { ref } from 'vue'
import { notifyError, notifySuccess } from '../../ui/notify'
import { copyToClipboard } from '../../utils/clipboard'

export function useDriveIssuedPassword() {
  const resetPwdInput = ref('')
  const showIssued = ref(false)
  const issuedPassword = ref('')

  function fillRandomPwd() {
    resetPwdInput.value = randomPassword(18)
  }

  function randomPassword(length: number): string {
    const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789'
    const arr = new Uint32Array(length)
    crypto.getRandomValues(arr)
    let s = ''
    for (let i = 0; i < length; i++) s += chars[arr[i]! % chars.length]
    return s
  }

  async function copyIssuedPassword() {
    try {
      await copyToClipboard(issuedPassword.value)
      notifySuccess('已复制', '密码已复制到剪贴板')
    } catch (e: any) {
      notifyError('复制失败', e?.message || String(e))
    }
  }

  return {
    resetPwdInput,
    showIssued,
    issuedPassword,
    fillRandomPwd,
    copyIssuedPassword,
  }
}
