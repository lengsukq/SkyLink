export type CfDnsRecord = Record<string, any>

export type CachedCfRecordsPayload = {
  records: CfDnsRecord[]
  updatedAt: number
}

