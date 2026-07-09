import { defineStore } from 'pinia'
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api, type Settings } from '@/api'

export const useSettingsStore = defineStore('settings', () => {
  const settings = ref<Settings>({
    close_to_tray: false,
    auto_start: false,
    log_retention_days: 0,
    config_dir: '',
    theme_mode: 'system',
  })
  const configDir = ref('')
  const loading = ref(false)

  async function load() {
    loading.value = true
    try {
      settings.value = await api.getSettings()
      // 旧配置缺 theme_mode 时按 system
      if (!settings.value.theme_mode) {
        settings.value.theme_mode = 'system'
      }
      configDir.value = await api.getConfigDir()
    } catch (e: any) {
      ElMessage.error('加载设置失败: ' + e.message)
    } finally {
      loading.value = false
    }
  }

  async function save(): Promise<boolean> {
    try {
      await api.saveSettings(settings.value)
      return true
    } catch (e: any) {
      ElMessage.error('保存设置失败: ' + e.message)
      return false
    }
  }

  // 单字段变更即时保存（开关类）
  async function saveField(field: keyof Settings): Promise<boolean> {
    return await save()
  }

  return { settings, configDir, loading, load, save, saveField }
})
