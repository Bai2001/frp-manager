import { defineStore } from 'pinia'
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api, type ServerInfo, type AddServerInput, type Capabilities } from '@/api'

export const useServerStore = defineStore('server', () => {
  const servers = ref<ServerInfo[]>([])
  const loading = ref(false)
  const capabilities = ref<Capabilities | null>(null)

  async function refresh() {
    loading.value = true
    try {
      servers.value = await api.listServers()
    } catch (e: any) {
      ElMessage.error('加载服务器列表失败: ' + e.message)
    } finally {
      loading.value = false
    }
  }

  async function addServer(input: AddServerInput): Promise<boolean> {
    try {
      await api.addServer(input)
      await refresh()
      ElMessage.success('服务器已添加')
      return true
    } catch (e: any) {
      ElMessage.error('添加失败: ' + e.message)
      return false
    }
  }

  async function updateServer(id: string, input: AddServerInput): Promise<boolean> {
    try {
      await api.updateServerByID(id, input)
      await refresh()
      ElMessage.success('服务器已更新')
      return true
    } catch (e: any) {
      ElMessage.error('更新失败: ' + e.message)
      return false
    }
  }

  async function deleteServer(id: string): Promise<boolean> {
    try {
      await api.deleteServer(id)
      await refresh()
      ElMessage.success('服务器已删除')
      return true
    } catch (e: any) {
      ElMessage.error('删除失败: ' + e.message)
      return false
    }
  }

  async function checkCapabilities(id: string): Promise<boolean> {
    try {
      capabilities.value = await api.checkServerCapabilities(id)
      ElMessage.success('服务端能力已获取')
      return true
    } catch (e: any) {
      capabilities.value = null
      ElMessage.error('检测能力失败: ' + e.message)
      return false
    }
  }

  return { servers, loading, capabilities, refresh, addServer, updateServer, deleteServer, checkCapabilities }
})
