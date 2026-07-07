import { defineStore } from 'pinia'
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api, type TunnelInfo, type AddTunnelInput } from '@/api'

export const useTunnelStore = defineStore('tunnel', () => {
  const tunnels = ref<TunnelInfo[]>([])
  const loading = ref(false)

  async function refresh(serverId?: string) {
    loading.value = true
    try {
      tunnels.value = await api.listTunnels(serverId)
    } catch (e: any) {
      ElMessage.error('加载映射列表失败: ' + e.message)
    } finally {
      loading.value = false
    }
  }

  async function addTunnel(input: AddTunnelInput): Promise<boolean> {
    try {
      await api.addTunnel(input)
      await refresh()
      ElMessage.success('映射已创建')
      return true
    } catch (e: any) {
      ElMessage.error('创建失败: ' + e.message)
      return false
    }
  }

  async function deleteTunnel(id: string): Promise<boolean> {
    try {
      await api.deleteTunnel(id)
      await refresh()
      ElMessage.success('映射已删除')
      return true
    } catch (e: any) {
      ElMessage.error('删除失败: ' + e.message)
      return false
    }
  }

  async function startFrpc(serverId: string): Promise<boolean> {
    try {
      await api.startFrpc(serverId)
      ElMessage.success('frpc 已启动')
      return true
    } catch (e: any) {
      ElMessage.error('启动失败: ' + e.message)
      return false
    }
  }

  async function stopFrpc(serverId: string): Promise<boolean> {
    try {
      await api.stopFrpc(serverId)
      ElMessage.success('frpc 已停止')
      return true
    } catch (e: any) {
      ElMessage.error('停止失败: ' + e.message)
      return false
    }
  }

  async function restartFrpc(serverId: string): Promise<boolean> {
    try {
      await api.restartFrpc(serverId)
      ElMessage.success('frpc 已重启')
      return true
    } catch (e: any) {
      ElMessage.error('重启失败: ' + e.message)
      return false
    }
  }

  async function isRunning(serverId: string): Promise<boolean> {
    try {
      return await api.isFrpcRunning(serverId)
    } catch {
      return false
    }
  }

  return { tunnels, loading, refresh, addTunnel, deleteTunnel, startFrpc, stopFrpc, restartFrpc, isRunning }
})
