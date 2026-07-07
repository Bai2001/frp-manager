<script setup lang="ts">
import { useLogStore } from '@/stores/log'

const store = useLogStore()
</script>

<template>
  <div class="page">
    <div class="page-header">
      <h2>日志</h2>
      <div>
        <el-button size="small" @click="store.clear">清空</el-button>
      </div>
    </div>

    <div class="log-view">
      <div v-for="(line, i) in store.lines" :key="i" class="log-line">
        <span class="log-time">{{ line.time }}</span>
        <span class="log-level" :class="line.level">[{{ line.level.toUpperCase() }}]</span>
        <span class="log-msg">{{ line.message }}</span>
      </div>
      <el-empty v-if="store.lines.length === 0" description="暂无日志" />
    </div>
  </div>
</template>

<style scoped>
.page { padding: 16px; height: 100%; display: flex; flex-direction: column; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h2 { margin: 0; }
.log-view {
  flex: 1; overflow: auto;
  background: #1e1e1e; border-radius: 4px; padding: 12px;
  font-family: Consolas, "Courier New", monospace; font-size: 12px;
}
.log-line { color: #d4d4d4; margin-bottom: 2px; }
.log-time { color: #888; margin-right: 8px; }
.log-level { margin-right: 8px; }
.log-level.info { color: #4fc3f7; }
.log-level.warn { color: #ffb74d; }
.log-level.error { color: #e57373; }
</style>
