import { ref } from 'vue'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { useEasyTierStatusQueries } from './useEasyTierStatusQueries'

const apiGetMock = vi.fn()
const notifySuccessMock = vi.fn()

vi.mock('../../api/client', () => ({
  default: {
    get: (...args: unknown[]) => apiGetMock(...args),
  },
}))

vi.mock('../../ui/notify', () => ({
  notifySuccess: (...args: unknown[]) => notifySuccessMock(...args),
}))

describe('useEasyTierStatusQueries', () => {
  beforeEach(() => {
    apiGetMock.mockReset()
    notifySuccessMock.mockReset()
  })

  it('loads status and cli raw outputs', async () => {
    apiGetMock
      .mockResolvedValueOnce({
        data: {
          ok: true,
          version: 'v1.2.3',
          self_ipv4: '10.0.0.2',
          self_hostname: 'node-1',
          peers: [{ ipv4: '10.0.0.3' }],
          routes: [{ ipv4: '10.0.0.0' }],
        },
      })
      .mockResolvedValueOnce({
        data: {
          stdout: '| IPv4 | Hostname | Cost | Lat(ms) | Loss | RX | TX | Tunnel | NAT | Version |\n|---|---|---|---|---|---|---|---|---|---|\n| 10.0.0.3 | node-2 | p2p | 12 | 0% | 1KB | 2KB | udp | fullcone | v1 |',
          stderr: '',
        },
      })

    const q = useEasyTierStatusQueries({
      profilePath: (p) => `/easytier/profiles/p1${p}`,
      imageTag: ref('latest'),
    })

    await q.loadStatus()
    await q.loadCliRaw()

    expect(q.status.ok).toBe(true)
    expect(q.status.version).toBe('v1.2.3')
    expect(q.status.peers.length).toBe(1)
    expect(q.parsedCliPeerRows.value.length).toBe(1)
  })

  it('checks updates and applies latest version', async () => {
    apiGetMock.mockResolvedValueOnce({
      data: {
        current_version: 'v1.0.0',
        latest_version: 'v1.1.0',
        update_available: true,
        release_url: 'https://example.com/release',
      },
    })

    const imageTag = ref('v1.0.0')
    const q = useEasyTierStatusQueries({
      profilePath: (p) => `/easytier${p}`,
      imageTag,
    })

    await q.checkUpdate()
    q.useLatestVersion()

    expect(q.versionCheck.latest_version).toBe('v1.1.0')
    expect(imageTag.value).toBe('v1.1.0')
    expect(notifySuccessMock).toHaveBeenCalledTimes(1)
  })
})
