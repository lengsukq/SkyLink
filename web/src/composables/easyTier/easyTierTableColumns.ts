import { h } from 'vue'
import { NTag } from 'naive-ui'
import type { CliPeerRow } from './cliRawParser'
import type { DisplayNodeRow } from './types'

export const nodeTableColumns = [
  { title: '虚拟IPv4地址', key: 'ipv4', width: 140, ellipsis: true },
  { title: '主机名', key: 'hostname', ellipsis: true },
  { title: '路由', key: 'route', width: 70 },
  { title: '协议', key: 'tunnel', width: 100, ellipsis: true },
  {
    title: '延迟',
    key: 'latency_ms',
    width: 80,
    render: (row: DisplayNodeRow) => (row.latency_ms != null ? `${Number(row.latency_ms)}ms` : '—'),
  },
  { title: '内核版本', key: 'version', width: 120, ellipsis: true },
]

export const routeColumns = [
  { title: 'IPv4', key: 'ipv4', width: 120 },
  { title: 'Hostname', key: 'hostname', ellipsis: true },
  { title: 'Proxy CIDRs', key: 'proxy_cidrs', ellipsis: true },
  { title: 'NextHop', key: 'next_hop_ipv4', width: 120 },
]

export const cliPeerColumns = [
  { title: 'IPv4', key: 'ipv4', width: 130, ellipsis: true },
  { title: '节点', key: 'hostname', ellipsis: true },
  {
    title: '链路',
    key: 'cost',
    width: 95,
    render: (row: CliPeerRow) =>
      h(
        NTag,
        { size: 'small', type: row.cost.toLowerCase().includes('relay') ? 'warning' : 'success', bordered: false },
        { default: () => row.cost }
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
  { title: '流量(RX/TX)', key: 'traffic', width: 150, render: (row: CliPeerRow) => `${row.rx} / ${row.tx}` },
  { title: 'Tunnel', key: 'tunnel', width: 90, ellipsis: true },
  { title: 'NAT', key: 'nat', width: 140, ellipsis: true },
  { title: '版本', key: 'version', width: 140, ellipsis: true },
]
