<script setup lang="ts">
import { onMounted, onUnmounted, ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useServerStore } from '@/stores/server'
import { useTunnelStore } from '@/stores/tunnel'
import { type AddTunnelInput } from '@/api'
import TunnelFormDialog from '@/components/TunnelFormDialog.vue'

const serverStore = useServerStore()
const tunnelStore = useTunnelStore()

const selectedServerId = ref('')
const dialogVisible = ref(false)
const runningMap = ref<Record<string, boolean>>({})
let pollTimer: number | undefined

const hasServer = computed(() => serverStore.servers.length > 0)

onMounted(async () => {
  await serverStore.refresh()
  if (serverStore.servers.length > 0) {
    selectedServerId.value = serverStore.servers[0].id
    await refreshTunnels()
    startPolling()
  }
})

onUnmounted(() => {
  if (pollTimer) window.clearInterval(pollTimer)
})

async function refreshTunnels() {
  if (selectedServerId.value) {
    await tunnelStore.refresh(selectedServerId.value)
    await pollRunning()
  }
}

function startPolling() {
  if (pollTimer) window.clearInterval(pollTimer)
  pollTimer = window.setInterval(pollRunning, 3000)
}

async function pollRunning() {
  for (const tu of tunnelStore.tunnels) {
    runningMap.value[tu.server_id] = await tunnelStore.isRunning(tu.server_id)
  }
}

function openCreate() {
  if (!selectedServerId.value) {
    ElMessage.warning('请先添加服务器')
    return
  }
  dialogVisible.value = true
}

async function handleSubmit(input: AddTunnelInput) {
  const ok = await tunnelStore.addTunnel(input)
  if (ok) {
    dialogVisible.value = false
    await refreshTunnels()
  }
}

async function handleDelete(id: string) {
  await ElMessageBox.confirm('确认删除该映射？将释放服务端资源。', '确认', { type: 'warning' })
  await tunnelStore.deleteTunnel(id)
  await refreshTunnels()
}

async function handleStart() {
  if (!selectedServerId.value) return
  await tunnelStore.startFrpc(selectedServerId.value)
  await pollRunning()
}

async function handleStop() {
  if (!selectedServerId.value) return
  await tunnelStore.stopFrpc(selectedServerId.value)
  await pollRunning()
}

async function handleRestart() {
  if (!selectedServerId.value) return
  await tunnelStore.restartFrpc(selectedServerId.value)
  await pollRunning()
}

async function onServerChange() {
  await refreshTunnels()
  startPolling()
}
</script>

<template>
  <div class="page">
    <div class="page-header">
      <h2>映射</h2>
      <div class="actions">
        <el-select v-model="selectedServerId" placeholder="选择服务器" @change="onServerChange" style="width: 180px">
          <el-option v-for="s in serverStore.servers" :key="s.id" :label="s.name" :value="s.id" />
        </el-select>
        <el-button type="primary" @click="openCreate" :disabled="!hasServer">创建映射</el-button>
        <el-button @click="handleStart" :disabled="!hasServer">启动 frpc</el-button>
        <el-button @click="handleStop" :disabled="!hasServer">停止 frpc</el-button>
        <el-button @click="handleRestart" :disabled="!hasServer">重启 frpc</el-button>
      </div>
    </div>

    <el-table :data="tunnelStore.tunnels" v-loading="tunnelStore.loading" border empty-text="暂无映射">
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="protocol" label="协议" width="80">
        <template #default="{ row }">
          <el-tag size="small">{{ row.protocol.toUpperCase() }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="本地">
        <template #default="{ row }">{{ row.local_ip }}:{{ row.local_port }}</template>
      </el-table-column>
      <el-table-column label="远程">
        <template #default="{ row }">
          <span v-if="row.remote_port">:{{ row.remote_port }}</span>
          <span v-else-if="row.custom_domain">{{ row.custom_domain }}</span>
          <span v-else-if="row.subdomain">{{ row.subdomain }}</span>
        </template>
      </el-table-column>
      <el-table-column label="frpc 状态" width="120">
        <template #default="{ row }">
          <el-tag :type="runningMap[row.server_id] ? 'success' : 'info'" size="small">
            {{ runningMap[row.server_id] ? '运行中' : '已停止' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="120">
        <template #default="{ row }">
          <el-button size="small" link type="danger" @click="handleDelete(row.id)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <TunnelFormDialog v-model:visible="dialogVisible" :server-id="selectedServerId" @submit="handleSubmit" />
  </div>
</template>

<style scoped>
.page { padding: 16px; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h2 { margin: 0; }
.actions { display: flex; gap: 8px; }
</style>
