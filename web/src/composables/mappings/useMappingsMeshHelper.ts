import { ref, type Ref } from 'vue'
import api from '../../api/client'
import type { EasyTierStatusAllResponse } from '../../types/mappings'

type MeshIpOption = { label: string; value: string }

type UseMappingsMeshHelperOptions = {
  addBackend: Ref<string>
  editBackend: Ref<string>
}

export function useMappingsMeshHelper(options: UseMappingsMeshHelperOptions) {
  const meshIpOptions = ref<MeshIpOption[]>([])
  const meshSelectedIp = ref<string | null>(null)
  const meshPort = ref<number>(3000)
  const meshSelectedIpEdit = ref<string | null>(null)
  const meshPortEdit = ref<number>(3000)

  async function loadMeshIps() {
    try {
      const { data } = await api.get<EasyTierStatusAllResponse>('/easytier/status/all')
      const ips: MeshIpOption[] = []
      ;(data?.profiles || []).forEach((item) => {
        const st = item?.status
        if (!item?.ok || !st) return
        if (st?.self_ipv4) ips.push({ label: `${st.self_ipv4} (${item?.name || item?.id || '本机'})`, value: st.self_ipv4 })
        ;(st?.peers || []).forEach((p) => {
          if (p.ipv4 && p.ipv4 !== st?.self_ipv4) ips.push({ label: `${p.ipv4} (${item?.name || item?.id || 'peer'})`, value: p.ipv4 })
        })
      })
      meshIpOptions.value = ips
    } catch (_) {
      meshIpOptions.value = []
    }
  }

  function fillBackendFromMesh(which: 'add' | 'edit') {
    const ip = which === 'add' ? meshSelectedIp.value : meshSelectedIpEdit.value
    const port = which === 'add' ? meshPort.value : meshPortEdit.value
    if (!ip) return
    const url = `http://${ip}:${port || 3000}`
    if (which === 'add') options.addBackend.value = url
    else options.editBackend.value = url
  }

  return {
    meshIpOptions,
    meshSelectedIp,
    meshPort,
    meshSelectedIpEdit,
    meshPortEdit,
    loadMeshIps,
    fillBackendFromMesh,
  }
}
