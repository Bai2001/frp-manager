// 封装对 Wails v3 后端 App 服务的调用。
// v3 通过 wails3 generate bindings 自动生成 TypeScript 绑定，
// 前端直接 import 调用，不再依赖 window.go 注入。

import * as AppService from '../../bindings/github.com/kdc/frp-manager/client/app'

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

// 应用设置，对应后端 settings.Settings（snake_case 匹配 JSON tag）
export interface Settings {
  close_to_tray: boolean
  auto_start: boolean
  log_retention_days: number
  config_dir: string
}

// 统一调用包装：v3 绑定方法返回 CancellablePromise，reject 时抛 Error。
// Wails reject 的值可能是字符串、对象或 Error，统一规范化成带 message 的 Error。
async function call<T>(fn: any, ...args: any[]): Promise<T> {
  if (!fn) {
    throw new Error('后端绑定未就绪')
  }
  try {
    return await fn(...args)
  } catch (e: any) {
    const msg = typeof e === 'string' ? e
      : (e?.message ?? e?.error ?? (typeof e === 'object' ? JSON.stringify(e) : String(e)))
    throw new Error(msg ?? '未知错误')
  }
}

export const api = {
  // 服务器
  async listServers(): Promise<ServerInfo[]> {
    return (await call(AppService.ListServers)) ?? []
  },
  async addServer(input: AddServerInput): Promise<string> {
    return await call(AppService.AddServer, input)
  },
  async updateServerByID(id: string, input: AddServerInput): Promise<void> {
    await call(AppService.UpdateServerByID, id, input)
  },
  async deleteServer(id: string): Promise<void> {
    await call(AppService.DeleteServer, id)
  },
  async checkServerCapabilities(id: string): Promise<Capabilities> {
    return await call(AppService.CheckServerCapabilities, id)
  },

  // 映射
  async listTunnels(serverId?: string): Promise<TunnelInfo[]> {
    return (await call(AppService.ListTunnels, serverId ?? '')) ?? []
  },
  async addTunnel(input: AddTunnelInput): Promise<string> {
    return await call(AppService.AddTunnel, input)
  },
  async updateTunnelByID(id: string, input: AddTunnelInput): Promise<void> {
    await call(AppService.UpdateTunnelByID, id, input)
  },
  async deleteTunnel(id: string): Promise<void> {
    await call(AppService.DeleteTunnel, id)
  },

  // frpc
  async generateFrpcConfig(serverId: string): Promise<string> {
    return await call(AppService.GenerateFrpcConfig, serverId)
  },
  async startFrpc(serverId: string): Promise<void> {
    await call(AppService.StartFrpc, serverId)
  },
  async stopFrpc(serverId: string): Promise<void> {
    await call(AppService.StopFrpc, serverId)
  },
  async restartFrpc(serverId: string): Promise<void> {
    await call(AppService.RestartFrpc, serverId)
  },
  async isFrpcRunning(serverId: string): Promise<boolean> {
    return await call(AppService.IsFrpcRunning, serverId)
  },

  // 设置 / 配置目录 / 导入导出
  async getSettings(): Promise<Settings> {
    return await call(AppService.GetSettings)
  },
  async saveSettings(s: Settings): Promise<void> {
    await call(AppService.SaveSettings, s)
  },
  async getConfigDir(): Promise<string> {
    return await call(AppService.GetConfigDir)
  },
  async openConfigDir(): Promise<void> {
    await call(AppService.OpenConfigDir)
  },
  async exportData(): Promise<string> {
    return await call(AppService.ExportData)
  },
  async importData(raw: string): Promise<void> {
    await call(AppService.ImportData, raw)
  },
}
