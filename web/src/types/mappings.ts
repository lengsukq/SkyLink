export interface MappingItem {
  id: number
  host: string
  backend: string
}

export interface MappingsListResponse {
  list: MappingItem[]
}

export interface CfZone {
  id: string
  name: string
}

export interface CfZonesResponse {
  zones: CfZone[]
}

export interface OneClickMappingRequest {
  host: string
  backend: string
  zone_id: string
  cname_target: string
}

export interface EasyTierStatusAllResponse {
  profiles: Array<{
    id: string
    name: string
    ok: boolean
    status?: {
      self_ipv4?: string
      peers?: Array<{ ipv4?: string }>
    }
  }>
}
