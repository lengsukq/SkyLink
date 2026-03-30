import { computed, ref, type Ref } from 'vue'
import api from '../../api/client'
import { notifyError } from '../../ui/notify'
import { getApiErrorMessage } from '../../utils/apiError'
import type { EasyTierProfile, EasyTierProfileCreateResponse, EasyTierProfilesResponse } from './types'

type UseEasyTierProfilesOptions = {
  imageTag: Ref<string>
  loadConfig: () => Promise<void>
  loadStatus: () => Promise<void>
  loadDaemonLogs: () => Promise<void>
}

export function useEasyTierProfiles(options: UseEasyTierProfilesOptions) {
  const profiles = ref<EasyTierProfile[]>([])
  const activeProfileId = ref('')
  const newProfileName = ref('')

  const profileOptions = computed(() => profiles.value.map((p) => ({ label: p.name || p.id, value: p.id })))
  const canDeleteProfile = computed(() => profiles.value.length > 1 && !!activeProfileId.value)

  function profilePath(path: string) {
    if (!activeProfileId.value) return `/easytier${path}`
    return `/easytier/profiles/${activeProfileId.value}${path}`
  }

  async function loadProfiles() {
    try {
      const { data } = await api.get<EasyTierProfilesResponse>('/easytier/profiles')
      profiles.value = data?.profiles || []
      activeProfileId.value = data?.active_profile_id || profiles.value[0]?.id || ''
    } catch (_) {
      profiles.value = []
      activeProfileId.value = ''
    }
  }

  async function onProfileChange(profileId: string) {
    if (!profileId) return
    try {
      await api.put(`/easytier/profiles/active/${profileId}`)
      await options.loadConfig()
      await options.loadStatus()
      await options.loadDaemonLogs()
    } catch (e) {
      notifyError('切换实例失败', getApiErrorMessage(e, '未知错误'))
    }
  }

  async function createProfile() {
    const name = (newProfileName.value || '').trim()
    if (!name) return
    try {
      const { data } = await api.post<EasyTierProfileCreateResponse>('/easytier/profiles', { name, config: { image_tag: options.imageTag.value || '' } })
      newProfileName.value = ''
      await loadProfiles()
      if (data?.id) {
        activeProfileId.value = data.id
        await onProfileChange(data.id)
      }
    } catch (e) {
      notifyError('新增实例失败', getApiErrorMessage(e, '未知错误'))
    }
  }

  async function deleteCurrentProfile() {
    if (!canDeleteProfile.value) return
    try {
      await api.delete(`/easytier/profiles/${activeProfileId.value}`)
      await loadProfiles()
      if (activeProfileId.value) {
        await onProfileChange(activeProfileId.value)
      }
    } catch (e) {
      notifyError('删除实例失败', getApiErrorMessage(e, '未知错误'))
    }
  }

  return {
    profiles,
    activeProfileId,
    newProfileName,
    profileOptions,
    canDeleteProfile,
    profilePath,
    loadProfiles,
    onProfileChange,
    createProfile,
    deleteCurrentProfile,
  }
}
