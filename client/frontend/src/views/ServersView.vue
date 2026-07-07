<script setup lang="ts">
import { onMounted } from 'vue'
import { useServerStore } from '@/stores/server'

const store = useServerStore()

onMounted(() => {
  store.refresh()
})
</script>

<template>
  <div class="page">
    <div class="page-header">
      <h2>服务器</h2>
      <el-button type="primary">添加服务器</el-button>
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
      <el-table-column label="操作" width="220">
        <template #default>
          <el-button size="small" link>检测能力</el-button>
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
