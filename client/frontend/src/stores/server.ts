import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api, type ServerInfo } from '@/api'

export const useServerStore = defineStore('server', () => {
  const servers = ref<ServerInfo[]>([])
  const loading = ref(false)

  async function refresh() {
    loading.value = true
    try {
      servers.value = await api.listServers()
    } finally {
      loading.value = false
    }
  }

  return { servers, loading, refresh }
})
