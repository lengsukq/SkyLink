export interface CliPeerRow {
  ipv4: string
  hostname: string
  cost: string
  latencyText: string
  latencyMs: number | null
  loss: string
  rx: string
  tx: string
  tunnel: string
  nat: string
  version: string
}

export interface CliPeerSummary {
  total: number
  p2p: number
  relay: number
  avgLatencyMs: number | null
}

export function parseCliPeerTable(stdout: string): CliPeerRow[] {
  const lines = stdout
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line.startsWith('|') && line.endsWith('|'))
  if (lines.length < 3) return []

  const headerLine = lines[0]
  const headerCells = splitMarkdownRow(headerLine).map((cell) => normalizeHeader(cell))
  if (!headerCells.length) return []

  const dataLines = lines.slice(2).filter((line) => !isSeparatorRow(line))
  const rows: CliPeerRow[] = []
  dataLines.forEach((line) => {
    const cells = splitMarkdownRow(line)
    if (!cells.length) return
    const picked = pickColumns(headerCells, cells)
    rows.push({
      ipv4: picked.ipv4 || '—',
      hostname: picked.hostname || '—',
      cost: picked.cost || '—',
      latencyText: picked.latency || '-',
      latencyMs: parseLatencyMs(picked.latency),
      loss: picked.loss || '-',
      rx: picked.rx || '-',
      tx: picked.tx || '-',
      tunnel: picked.tunnel || '-',
      nat: picked.nat || '-',
      version: picked.version || '-',
    })
  })

  return rows
}

export function getCliPeerSummary(rows: CliPeerRow[]): CliPeerSummary {
  if (!rows.length) {
    return { total: 0, p2p: 0, relay: 0, avgLatencyMs: null }
  }
  const p2p = rows.filter((row) => row.cost.toLowerCase().includes('p2p')).length
  const relay = rows.filter((row) => row.cost.toLowerCase().includes('relay')).length
  const latencyList = rows
    .map((row) => row.latencyMs)
    .filter((value): value is number => value !== null)
  const avgLatencyMs = latencyList.length
    ? Number((latencyList.reduce((sum, value) => sum + value, 0) / latencyList.length).toFixed(2))
    : null
  return {
    total: rows.length,
    p2p,
    relay,
    avgLatencyMs,
  }
}

function splitMarkdownRow(line: string): string[] {
  return line
    .split('|')
    .slice(1, -1)
    .map((cell) => cell.trim())
}

function isSeparatorRow(line: string): boolean {
  const cells = splitMarkdownRow(line)
  if (!cells.length) return false
  return cells.every((cell) => /^:?-{3,}:?$/.test(cell))
}

function normalizeHeader(raw: string): string {
  return raw.toLowerCase().replace(/\s+/g, '')
}

function pickColumns(headers: string[], cells: string[]) {
  const getBy = (matcher: (header: string) => boolean) => {
    const idx = headers.findIndex(matcher)
    if (idx < 0) return ''
    return (cells[idx] || '').trim()
  }
  return {
    ipv4: getBy((header) => header === 'ipv4'),
    hostname: getBy((header) => header === 'hostname'),
    cost: getBy((header) => header === 'cost'),
    latency: getBy((header) => header === 'lat(ms)' || header === 'latency' || header.includes('lat')),
    loss: getBy((header) => header === 'loss'),
    rx: getBy((header) => header === 'rx'),
    tx: getBy((header) => header === 'tx'),
    tunnel: getBy((header) => header === 'tunnel'),
    nat: getBy((header) => header === 'nat'),
    version: getBy((header) => header === 'version'),
  }
}

function parseLatencyMs(text: string): number | null {
  const normalized = (text || '').trim()
  if (!normalized || normalized === '-') return null
  const matched = normalized.match(/(\d+(?:\.\d+)?)/)
  if (!matched) return null
  const num = Number(matched[1])
  return Number.isFinite(num) ? num : null
}
