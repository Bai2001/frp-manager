// 封装对 Wails 后端 App 方法的调用。
// Wails v2 把 main 包里 Bind 的对象注入到 window.go.main.App 下。

export interface ServerInfo {
  id: string
  name: string
  host: string
  frps_port: number
  frp_token: string
  agent_url: string
  agent_token: string
  is_default: boolean
  remark?: string
}

export interface TunnelInfo {
  id: string
  server_id: string
  name: string
  protocol: 'tcp' | 'udp' | 'http' | 'https'
  local_ip: string
  local_port: number
  remote_port?: number
  custom_domain?: string
  subdomain?: string
  enabled: boolean
  status: string
}

export interface AddServerInput {
  name: string
  host: string
  frps_port: number
  frp_token: string
  agent_url: string
  agent_token: string
  is_default?: boolean
  remark?: string
}

export interface AddTunnelInput {
  server_id: string
  name: string
  protocol: 'tcp' | 'udp' | 'http' | 'https'
  local_ip: string
  local_port: number
  remote_port?: number
  custom_domain?: string
  subdomain?: string
}

export interface Capabilities {
  frps_running: boolean
  frps_version: string
  bind_port: number
  allow_ports: { start: number; end: number }[]
  support_tcp: boolean
  support_udp: boolean
  support_http: boolean
  support_https: boolean
  vhost_http_port: number
  vhost_https_port: number
  subdomain_host: string
  allowed_root_domains: string[]
}

export interface PortCheckResult {
  protocol: string
  port: number
  available: boolean
  reason: string
}

export interface DomainCheckResult {
  domain: string
  available: boolean
  reason: string
}

// Wails 绑定的后端对象。开发模式下 wails dev 会注入；类型宽松处理。
function backend(): any {
  return (window as any).go?.main?.App
}

// 统一调用包装：Wails 方法返回 Promise，reject 时抛 Error。
async function call<T>(fn: any, ...args: any[]): Promise<T> {
  if (!fn) {
    throw new Error('后端未就绪（window.go.main.App 不存在）')
  }
  return await fn(...args)
}

export const api = {
  // 服务器
  async listServers(): Promise<ServerInfo[]> {
    return (await call(backend()?.ListServers)) ?? []
  },
  async addServer(input: AddServerInput): Promise<string> {
    return await call(backend()?.AddServer, input)
  },
  async updateServerByID(id: string, input: AddServerInput): Promise<void> {
    await call(backend()?.UpdateServerByID, id, input)
  },
  async deleteServer(id: string): Promise<void> {
    await call(backend()?.DeleteServer, id)
  },
  async checkServerCapabilities(id: string): Promise<Capabilities> {
    return await call(backend()?.CheckServerCapabilities, id)
  },

  // 映射
  async listTunnels(serverId?: string): Promise<TunnelInfo[]> {
    return (await call(backend()?.ListTunnels, serverId ?? '')) ?? []
  },
  async addTunnel(input: AddTunnelInput): Promise<string> {
    return await call(backend()?.AddTunnel, input)
  },
  async updateTunnelByID(id: string, input: AddTunnelInput): Promise<void> {
    await call(backend()?.UpdateTunnelByID, id, input)
  },
  async deleteTunnel(id: string): Promise<void> {
    await call(backend()?.DeleteTunnel, id)
  },

  // frpc
  async generateFrpcConfig(serverId: string): Promise<string> {
    return await call(backend()?.GenerateFrpcConfig, serverId)
  },
  async startFrpc(serverId: string): Promise<void> {
    await call(backend()?.StartFrpc, serverId)
  },
  async stopFrpc(serverId: string): Promise<void> {
    await call(backend()?.StopFrpc, serverId)
  },
  async restartFrpc(serverId: string): Promise<void> {
    await call(backend()?.RestartFrpc, serverId)
  },
  async isFrpcRunning(serverId: string): Promise<boolean> {
    return await call(backend()?.IsFrpcRunning, serverId)
  },
}
