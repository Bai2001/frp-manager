// 封装对 Wails 后端 App 方法的调用。
// 后端通过 wails Bind 暴露的方法会注入到 window.go.<pkg>.<Struct> 下。
// 骨架阶段只声明接口与空实现，待后端方法实现后对接。

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

// 动态获取 wails 绑定的后端对象；未绑定时返回 undefined。
function backend(): any {
  return (window as any).go?.['github.com/kdc/frp-manager/client']?.App
}

export const api = {
  async listServers(): Promise<ServerInfo[]> {
    return (await backend()?.ListServers()) ?? []
  },
  async listTunnels(serverId?: string): Promise<TunnelInfo[]> {
    return (await backend()?.ListTunnels(serverId)) ?? []
  },
  async generateFrpcConfig(serverId: string): Promise<string> {
    return (await backend()?.GenerateFrpcConfig(serverId)) ?? ''
  },
  async startFrpc(serverId: string): Promise<void> {
    await backend()?.StartFrpc(serverId)
  },
  async stopFrpc(serverId: string): Promise<void> {
    await backend()?.StopFrpc(serverId)
  },
  async restartFrpc(serverId: string): Promise<void> {
    await backend()?.RestartFrpc(serverId)
  },
}
