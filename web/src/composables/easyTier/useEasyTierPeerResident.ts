import { h, ref, type Ref } from 'vue'
import { NTag } from 'naive-ui'
import api from '../../api/client'
import { parseCliPeerTable, type CliPeerRow } from './cliRawParser'
import type { EasyTierCliOutputResponse } from './types'

type UseEasyTierPeerResidentOptions = {
  easytierHostSupported: Ref<boolean>
  profilePath: (path: string) => string
}

export function useEasyTierPeerResident(options: UseEasyTierPeerResidentOptions) {
  const peerResidentRows = ref<CliPeerRow[]>([])
  const peerResidentColumns = [
    { title: '虚拟IPv4地址', key: 'ipv4', width: 140, ellipsis: true },
    { title: '节点', key: 'hostname', ellipsis: true },
    {
      title: '链路',
      key: 'cost',
      width: 95,
      render: (row: CliPeerRow) =>
        h(
          NTag,
          { size: 'small', type: row.cost?.toLowerCase?.().includes('relay') ? 'warning' : 'success', bordered: false },
          { default: () => row.cost || '—' }
        ),
    },
    {
      title: '延迟',
      key: 'latencyText',
      width: 90,
      render: (row: CliPeerRow) => {
        if (row.latencyMs == null) return '—'
        const latencyMs = row.latencyMs
        return h(
          NTag,
          { size: 'small', type: latencyMs > 300 ? 'warning' : 'info', bordered: false },
          { default: () => `${latencyMs.toFixed(2)} ms` }
        )
      },
    },
    { title: '丢包', key: 'loss', width: 80 },
    { title: '流量(RX/TX)', key: 'traffic', width: 150, render: (row: CliPeerRow) => `${row.rx || '-'} / ${row.tx || '-'}` },
    { title: 'Tunnel', key: 'tunnel', width: 90, ellipsis: true },
    { title: 'NAT', key: 'nat', width: 140, ellipsis: true },
    { title: '版本', key: 'version', width: 140, ellipsis: true },
  ]

  let statusPollTimer: ReturnType<typeof setInterval> | null = null
  let statusPollInFlight = false

  async function loadPeerResidentOnly() {
    try {
      const { data } = await api.get<EasyTierCliOutputResponse>(options.profilePath('/cli-output'), {
        params: { target: 'peer' },
        silentError: true,
      })
      const stdout = typeof data?.stdout === 'string' ? data.stdout : ''
      peerResidentRows.value = parseCliPeerTable(stdout)
    } catch (_) {
      peerResidentRows.value = []
    }
  }

  function startStatusPolling() {
    stopStatusPolling()
    statusPollTimer = setInterval(async () => {
      if (statusPollInFlight) return
      if (!options.easytierHostSupported.value) return
      if (typeof document !== 'undefined' && document.visibilityState !== 'visible') return
      statusPollInFlight = true
      try {
        await loadPeerResidentOnly()
      } finally {
        statusPollInFlight = false
      }
    }, 1000)
  }

  function stopStatusPolling() {
    if (!statusPollTimer) return
    clearInterval(statusPollTimer)
    statusPollTimer = null
  }

  return {
    peerResidentRows,
    peerResidentColumns,
    loadPeerResidentOnly,
    startStatusPolling,
    stopStatusPolling,
  }
}
