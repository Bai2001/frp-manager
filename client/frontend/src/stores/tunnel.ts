import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api, type TunnelInfo } from '@/api'

export const useTunnelStore = defineStore('tunnel', () => {
  const tunnels = ref<TunnelInfo[]>([])
  const loading = ref(false)

  async function refresh(serverId?: string) {
    loading.value = true
    try {
      tunnels.value = await api.listTunnels(serverId)
    } finally {
      loading.value = false
    }
  }

  return { tunnels, loading, refresh }
})
