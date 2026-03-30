import api from '../../api/client'
import type { EasyTierRuntimeInstallResponse, RuntimeInstalledItem } from './types'

export async function installEasyTierRuntime(version: string, osVal: string, arch: string) {
  const { data } = await api.post<EasyTierRuntimeInstallResponse>('/easytier/runtime/install', {
    version,
    os: osVal,
    arch,
  })
  return data
}

export async function removeEasyTierRuntime(version: string, osVal: string, arch: string) {
  await api.delete('/easytier/runtime', {
    data: { version, os: osVal, arch },
  })
}

export async function removeInstalledRuntime(row: RuntimeInstalledItem) {
  await removeEasyTierRuntime(row.version, row.os, row.arch)
}

