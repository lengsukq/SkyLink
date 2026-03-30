import api from '../../api/client'
import type {
  EasyTierPlatformResponse,
  EasyTierPlatformsResponse,
  EasyTierReleasesResponse,
  RuntimePlatformItem,
  RuntimeReleaseItem,
} from './types'

export async function fetchEasyTierPlatform() {
  const { data } = await api.get<EasyTierPlatformResponse>('/easytier/platform')
  return {
    os: data?.os || '',
    arch: data?.arch || '',
    label: data?.label || '',
    easytierHostSupported: data?.easytier_host_supported === true,
  }
}

export async function fetchEasyTierReleases() {
  const { data } = await api.get<EasyTierReleasesResponse>('/easytier/releases')
  const list: RuntimeReleaseItem[] = data?.releases || []
  return list
}

export async function fetchEasyTierPlatforms() {
  const { data } = await api.get<EasyTierPlatformsResponse>('/easytier/platforms')
  return {
    list: (data?.platforms || []) as RuntimePlatformItem[],
    currentLabel: data?.current?.label || '',
    easytierHostSupported: data?.easytier_host_supported,
  }
}

