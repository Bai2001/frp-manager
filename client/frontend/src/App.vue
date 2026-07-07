<script lang="ts" setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const active = computed(() => route.path)

const menus = [
  { index: '/servers', label: '服务器', icon: 'Connection' },
  { index: '/tunnels', label: '映射', icon: 'Position' },
  { index: '/logs', label: '日志', icon: 'Document' },
  { index: '/settings', label: '设置', icon: 'Setting' },
]
</script>

<template>
  <el-container class="layout">
    <el-aside width="200px" class="aside">
      <div class="brand">FRP Manager</div>
      <el-menu :default-active="active" @select="(i: string) => router.push(i)">
        <el-menu-item v-for="m in menus" :key="m.index" :index="m.index">
          {{ m.label }}
        </el-menu-item>
      </el-menu>
    </el-aside>
    <el-main class="main">
      <router-view />
    </el-main>
  </el-container>
</template>

<style scoped>
.layout { height: 100vh; }
.aside { background: #fff; border-right: 1px solid #e4e7ed; }
.brand {
  height: 56px; line-height: 56px; text-align: center;
  font-size: 16px; font-weight: 600; color: #303133;
  border-bottom: 1px solid #e4e7ed;
}
.main { padding: 0; background: #f5f7fa; overflow: auto; }
</style>
