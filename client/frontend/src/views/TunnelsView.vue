<script setup lang="ts">
import { onMounted } from 'vue'
import { useTunnelStore } from '@/stores/tunnel'

const store = useTunnelStore()

onMounted(() => {
  store.refresh()
})
</script>

<template>
  <div class="page">
    <div class="page-header">
      <h2>映射</h2>
      <el-button type="primary">创建映射</el-button>
    </div>

    <el-table :data="store.tunnels" v-loading="store.loading" border empty-text="暂无映射">
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
      <el-table-column prop="status" label="状态" width="100" />
      <el-table-column label="操作" width="200">
        <template #default>
          <el-button size="small" link>启动</el-button>
          <el-button size="small" link>编辑</el-button>
          <el-button size="small" link type="danger">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<style scoped>
.page { padding: 16px; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h2 { margin: 0; }
</style>
