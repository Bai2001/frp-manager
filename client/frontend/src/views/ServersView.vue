<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessageBox } from 'element-plus'
import { useServerStore } from '@/stores/server'
import { type AddServerInput, type ServerInfo } from '@/api'
import ServerFormDialog from '@/components/ServerFormDialog.vue'

const store = useServerStore()
const dialogVisible = ref(false)
const editing = ref<ServerInfo | null>(null)

onMounted(() => {
  store.refresh()
})

function openAdd() {
  editing.value = null
  dialogVisible.value = true
}

function openEdit(row: ServerInfo) {
  editing.value = row
  dialogVisible.value = true
}

async function handleSubmit(input: AddServerInput) {
  let ok: boolean
  if (editing.value) {
    ok = await store.updateServer(editing.value.id, input)
  } else {
    ok = await store.addServer(input)
  }
  if (ok) dialogVisible.value = false
}

async function handleDelete(row: ServerInfo) {
  await ElMessageBox.confirm(`确认删除服务器「${row.name}」？其下映射将一并删除。`, '确认', { type: 'warning' })
  await store.deleteServer(row.id)
}

async function handleCheckCapabilities(row: ServerInfo) {
  await store.checkCapabilities(row.id)
}
</script>

<template>
  <div class="page">
    <div class="page-header">
      <h2>服务器</h2>
      <el-button type="primary" @click="openAdd">添加服务器</el-button>
    </div>

    <el-table :data="store.servers" v-loading="store.loading" border empty-text="暂无服务器">
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="host" label="公网地址" />
      <el-table-column prop="frps_port" label="frps 端口" width="100" />
      <el-table-column prop="agent_url" label="Agent 地址" />
      <el-table-column prop="is_default" label="默认" width="80">
        <template #default="{ row }">
          <el-tag v-if="row.is_default" type="success" size="small">默认</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="260">
        <template #default="{ row }">
          <el-button size="small" link @click="handleCheckCapabilities(row)">检测能力</el-button>
          <el-button size="small" link @click="openEdit(row)">编辑</el-button>
          <el-button size="small" link type="danger" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <ServerFormDialog v-model:visible="dialogVisible" :editing="editing" @submit="handleSubmit" />

    <el-dialog v-model="store.capabilities" title="服务端能力" width="560px" v-if="store.capabilities">
      <el-descriptions :column="2" border>
        <el-descriptions-item label="frps 运行">{{ store.capabilities.frps_running ? '是' : '否' }}</el-descriptions-item>
        <el-descriptions-item label="frps 版本">{{ store.capabilities.frps_version || '-' }}</el-descriptions-item>
        <el-descriptions-item label="bind 端口">{{ store.capabilities.bind_port }}</el-descriptions-item>
        <el-descriptions-item label="HTTP vhost">{{ store.capabilities.vhost_http_port || '-' }}</el-descriptions-item>
        <el-descriptions-item label="HTTPS vhost">{{ store.capabilities.vhost_https_port || '-' }}</el-descriptions-item>
        <el-descriptions-item label="子域名根">{{ store.capabilities.subdomain_host || '-' }}</el-descriptions-item>
        <el-descriptions-item label="支持协议">
          <el-tag v-if="store.capabilities.support_tcp" size="small">TCP</el-tag>
          <el-tag v-if="store.capabilities.support_udp" size="small">UDP</el-tag>
          <el-tag v-if="store.capabilities.support_http" size="small">HTTP</el-tag>
          <el-tag v-if="store.capabilities.support_https" size="small">HTTPS</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="允许根域名">{{ store.capabilities.allowed_root_domains.join(', ') }}</el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<style scoped>
.page { padding: 16px; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h2 { margin: 0; }
</style>
