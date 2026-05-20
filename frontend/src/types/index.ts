export interface Group {
  id: string
  name: string
  parent_id: string | null
  sort_order: number
}

export type AuthType = 'password' | 'private_key' | 'agent' | 'interactive'

export interface Connection {
  id: string
  group_id: string | null
  name: string
  host: string
  port: number
  username: string
  auth_type: AuthType
  upload_path: string
  default_cmd: string | null
  sort_order: number
  color: string | null
  last_used_at: string | null
}

export interface ConnectionForm {
  id: string
  group_id: string | null
  name: string
  host: string
  port: number
  username: string
  auth_type: AuthType
  password?: string
  private_key?: string
  key_passphrase?: string
  proxy_type?: string
  proxy_addr?: string
  jump_host_id?: string
  upload_path: string
  default_cmd?: string
  sort_order: number
  color?: string
}

export interface QuickCommand {
  id: string
  name: string
  command: string
  connection_id: string | null
  sort_order: number
}

export type ForwardType = 'local' | 'remote' | 'dynamic'

export interface PortForward {
  id: string
  name: string
  connection_id: string
  connection_name: string
  connection_host: string
  type: ForwardType
  local_host: string
  local_port: number
  remote_host: string
  remote_port: number
  auto_start: boolean
}

export interface SystemStats {
  cpu_percent: number
  mem_percent: number
  mem_total: number
  mem_used: number
  net_interfaces: Record<string, NetIO>
  load_avg: [number, number, number]
  disk_stats: DiskStat[]
  uptime_seconds: number
  os: string
}

export interface NetIO {
  receive_bytes: number
  transmit_bytes: number
  receive_kbps: number
  transmit_kbps: number
}

export interface DiskStat {
  device: string
  mount_point: string
  total: number
  used: number
  percent: number
}

export interface SFTPFileInfo {
  name: string
  size: number
  mode: number
  mod_time: number
  is_dir: boolean
}

export interface TransferProgress {
  id: string
  file_name: string
  total_bytes: number
  transferred: number
  percent: number
  speed_kbps: number
  done: boolean
  error?: string
}

export type SplitDirection = 'h' | 'v'

export interface SplitNode {
  type: 'split'
  direction: SplitDirection
  children: TreeNode[]
  ratio: number
}

export interface LeafNode {
  type: 'leaf'
  sessionID: string
}

export type TreeNode = SplitNode | LeafNode

export interface SSHKeyInfo {
  name: string
  type: string
  fingerprint: string
  public_key: string
  comment: string
  has_passphrase: boolean
}

export interface SSHConfigDirective {
  key: string
  value: string
}

export interface SSHConfigEntry {
  type: string
  pattern: string
  directives: SSHConfigDirective[]
}

export interface SSHConfigImportCandidate {
  pattern: string
  hostname: string
  port: number
  user: string
  identity_file: string
  has_key: boolean
}
