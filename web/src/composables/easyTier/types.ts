export interface EasyTierFormState {
  network_name: string
  network_secret: string
  peers: string
  ipv4: string
  enabled: boolean
  image_tag: string
  rpc_portal: string
  hostname: string
  external_node: string
  proxy_networks: string
  dhcp: boolean
}

export interface EasyTierStatusPeer {
  ipv4?: string
  hostname?: string
  tunnel?: string
  latency_ms?: number | null
  version?: string
}

export interface EasyTierStatusRoute {
  ipv4?: string
  hostname?: string
  proxy_cidrs?: string
  next_hop_ipv4?: string
}

export interface EasyTierStatusState {
  ok: boolean
  error: string
  hint: string
  version: string
  self_ipv4: string
  self_hostname: string
  peers: EasyTierStatusPeer[]
  routes: EasyTierStatusRoute[]
}

export interface EasyTierProfile {
  id: string
  name?: string
}

export interface RuntimeReleaseItem {
  tag_name: string
}

export interface RuntimePlatformItem {
  os: string
  arch: string
  label?: string
}

export interface RuntimeInstalledItem {
  version: string
  os: string
  arch: string
}

export interface SelectOption {
  label: string
  value: string
}

export interface DisplayNodeRow {
  ipv4: string
  hostname: string
  route: string
  tunnel: string
  latency_ms: number | null
  version: string
}

export interface EasyTierConfigResponse extends EasyTierFormState {}

export interface EasyTierProfilesResponse {
  profiles: EasyTierProfile[]
  active_profile_id?: string
}

export interface EasyTierProfileCreateResponse {
  id?: string
}

export interface EasyTierStatusResponse extends EasyTierStatusState {}

export interface EasyTierVersionCheckResponse {
  current_version?: string
  latest_version?: string
  update_available?: boolean
  release_url?: string
}

export interface EasyTierSettingsResponse {
  autostart_on_startup?: boolean
}

export interface EasyTierCliOutputResponse {
  ok?: boolean
  hint?: string
  stdout?: string
  stderr?: string
  error?: string
}

export interface EasyTierPlatformResponse {
  os?: string
  arch?: string
  label?: string
  easytier_host_supported?: boolean
}

export interface EasyTierPlatformsResponse {
  platforms?: RuntimePlatformItem[]
  current?: { label?: string }
  easytier_host_supported?: boolean
}

export interface EasyTierReleasesResponse {
  releases?: RuntimeReleaseItem[]
}

export interface EasyTierRuntimeInstalledResponse {
  installed?: boolean
}

export interface EasyTierRuntimeListResponse {
  items?: RuntimeInstalledItem[]
}

export interface EasyTierRuntimeInstallResponse {
  installed?: boolean
  version?: string
}

export interface EasyTierDaemonStatusResponse {
  running?: boolean
  pid?: number
  started_version?: string
  daemon_mode_enabled?: boolean
}

export interface EasyTierDaemonLogsResponse {
  logs?: string
}

export interface EasyTierDaemonActionResponse {
  message?: string
}

export interface EasyTierReleasePortResponse extends EasyTierDaemonActionResponse {
  killed?: number
  ports_freed?: number[]
}
