<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useSettingsStore } from '@/stores/settings'
import { api } from '@/api'

const store = useSettingsStore()
const saving = ref(false)
const importing = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

onMounted(() => {
  store.load()
})

async function onSwitchChange() {
  saving.value = true
  try {
    await store.save()
    ElMessage.success('设置已保存')
  } finally {
    saving.value = false
  }
}

async function onLogRetentionChange() {
  saving.value = true
  try {
    await store.save()
    ElMessage.success('日志保留设置已保存')
  } finally {
    saving.value = false
  }
}

async function openDir() {
  try {
    await api.openConfigDir()
  } catch (e: any) {
    ElMessage.error('打开目录失败: ' + e.message)
  }
}

async function exportData() {
  try {
    const raw = await api.exportData()
    const blob = new Blob([raw], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    const ts = new Date().toISOString().slice(0, 10)
    a.download = `frp-manager-backup-${ts}.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    ElMessage.success('数据已导出')
  } catch (e: any) {
    ElMessage.error('导出失败: ' + e.message)
  }
}

function triggerImport() {
  fileInput.value?.click()
}

async function onFileSelected(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  importing.value = true
  try {
    const text = await file.text()
    await ElMessageBox.confirm(
      '导入将清空并替换当前所有服务器和映射数据，确定继续？',
      '导入确认',
      { type: 'warning', confirmButtonText: '确定导入', cancelButtonText: '取消' },
    )
    await api.importData(text)
    ElMessage.success('数据已导入，请刷新服务器/映射列表')
  } catch (e: any) {
    if (e !== 'cancel' && e?.message !== 'cancel') {
      ElMessage.error('导入失败: ' + (e?.message ?? e))
    }
  } finally {
    importing.value = false
    input.value = '' // 允许重复导入同一文件
  }
}
</script>

<template>
  <div class="page">
    <h2>设置</h2>

    <el-form label-width="160px" style="max-width: 560px" v-loading="store.loading">
      <el-form-item label="关闭时最小化到托盘">
        <el-switch v-model="store.settings.close_to_tray" :loading="saving" @change="onSwitchChange" />
      </el-form-item>

      <el-form-item label="开机自启">
        <el-switch v-model="store.settings.auto_start" :loading="saving" @change="onSwitchChange" />
      </el-form-item>

      <el-form-item label="日志保留">
        <el-input-number v-model="store.settings.log_retention_days" :min="0" :max="30" :loading="saving" @change="onLogRetentionChange" />
        <span class="hint">天（0 = 不落盘，仅内存）</span>
      </el-form-item>

      <el-divider content-position="left">数据目录</el-divider>

      <el-form-item label="配置目录">
        <el-input :model-value="store.configDir" readonly placeholder="未指定">
          <template #append>
            <el-button @click="openDir">打开目录</el-button>
          </template>
        </el-input>
        <div class="hint-block">当前仅展示，修改需迁移数据，暂不支持。</div>
      </el-form-item>

      <el-divider content-position="left">导入 / 导出</el-divider>

      <el-form-item label="备份与恢复">
        <el-button @click="exportData">导出数据</el-button>
        <el-button @click="triggerImport" :loading="importing">导入数据</el-button>
        <input ref="fileInput" type="file" accept=".json,application/json" style="display:none" @change="onFileSelected" />
      </el-form-item>
    </el-form>
  </div>
</template>

<style scoped>
.page { padding: 16px; }
.page h2 { margin: 0 0 16px 0; }
.hint { margin-left: 8px; color: #909399; }
.hint-block { margin-top: 4px; font-size: 12px; color: #909399; }
</style>
