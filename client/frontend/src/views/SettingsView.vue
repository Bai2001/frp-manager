<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useSettingsStore } from '@/stores/settings'
import { api, type ThemeMode } from '@/api'
import { watchTheme } from '@/composables/useTheme'

const store = useSettingsStore()
const saving = ref(false)
const importing = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

onMounted(() => {
    store.load().then(() => {
        watchTheme(store.settings.theme_mode)
    })
})

async function onSwitchChange() {
    saving.value = true
    try {
        await store.save()
    } catch (e: any) {
        ElMessage.error('保存设置失败: ' + e.message)
    } finally {
        saving.value = false
    }
}

/**
 * 外观主题变更：立即应用并持久化
 */
async function onThemeChange(val: string | number | boolean | undefined) {
    const mode = (
        val === 'light' || val === 'dark' || val === 'system' ? val : 'system'
    ) as ThemeMode
    store.settings.theme_mode = mode
    watchTheme(mode)
    saving.value = true
    try {
        await store.save()
    } catch (e: any) {
        ElMessage.error('保存外观主题失败: ' + e.message)
    } finally {
        saving.value = false
    }
}

async function onLogRetentionChange() {
    saving.value = true
    try {
        await store.save()
    } catch (e: any) {
        ElMessage.error('保存日志保留设置失败: ' + e.message)
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
        const savedPath = await api.exportDataToPath()
        if (!savedPath) {
            // 用户取消
            return
        }
        ElMessage.success(`已导出到：${savedPath}`)
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
            {
                type: 'warning',
                confirmButtonText: '确定导入',
                cancelButtonText: '取消',
            }
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
        <div class="page-title">
            <h2>设置</h2>
            <p class="page-desc">应用偏好与数据管理</p>
        </div>

        <div class="cards" v-loading="store.loading">
            <!-- 通用设置 -->
            <el-card class="setting-card" shadow="never">
                <template #header>
                    <div class="card-header">
                        <span class="card-title">通用</span>
                        <span class="card-sub">应用行为偏好</span>
                    </div>
                </template>
                <el-form label-width="160px" class="setting-form">
                    <el-form-item label="外观主题">
                        <el-radio-group
                            :model-value="store.settings.theme_mode || 'system'"
                            :disabled="saving"
                            @change="onThemeChange"
                        >
                            <el-radio-button value="system">跟随系统</el-radio-button>
                            <el-radio-button value="light">浅色</el-radio-button>
                            <el-radio-button value="dark">深色</el-radio-button>
                        </el-radio-group>
                    </el-form-item>
                    <el-form-item label="关闭时最小化到托盘">
                        <el-switch
                            v-model="store.settings.close_to_tray"
                            :loading="saving"
                            @change="onSwitchChange"
                        />
                    </el-form-item>
                    <el-form-item label="开机自启">
                        <el-switch
                            v-model="store.settings.auto_start"
                            :loading="saving"
                            @change="onSwitchChange"
                        />
                    </el-form-item>
                    <el-form-item label="日志保留">
                        <el-input-number
                            v-model="store.settings.log_retention_days"
                            :min="0"
                            :max="30"
                            :loading="saving"
                            @change="onLogRetentionChange"
                        />
                        <span class="hint">天（0 = 不落盘，仅内存）</span>
                    </el-form-item>
                </el-form>
            </el-card>

            <!-- 数据目录 -->
            <el-card class="setting-card" shadow="never">
                <template #header>
                    <div class="card-header">
                        <span class="card-title">数据目录</span>
                        <span class="card-sub">配置文件存储位置</span>
                    </div>
                </template>
                <el-form label-width="160px" class="setting-form">
                    <el-form-item label="配置目录">
                        <el-input :model-value="store.configDir" readonly placeholder="未指定">
                            <template #append>
                                <el-button @click="openDir">打开目录</el-button>
                            </template>
                        </el-input>
                        <div class="hint-block">当前仅展示，修改需迁移数据，暂不支持。</div>
                    </el-form-item>
                </el-form>
            </el-card>

            <!-- 导入导出 -->
            <el-card class="setting-card" shadow="never">
                <template #header>
                    <div class="card-header">
                        <span class="card-title">备份与恢复</span>
                        <span class="card-sub">导入或导出应用数据</span>
                    </div>
                </template>
                <div class="backup-actions">
                    <el-button @click="exportData">导出数据</el-button>
                    <el-button @click="triggerImport" :loading="importing">导入数据</el-button>
                    <input
                        ref="fileInput"
                        type="file"
                        accept=".json,application/json"
                        style="display: none"
                        @change="onFileSelected"
                    />
                </div>
            </el-card>
        </div>
    </div>
</template>

<style scoped>
.page {
    padding: 24px;
    max-width: 960px;
    margin: 0 auto;
    flex: 1;
    min-height: 0;
}

.page-title h2 {
    margin: 0 0 4px 0;
    font-size: 20px;
    font-weight: 600;
    color: var(--content-fg);
}
.page-desc {
    margin: 0 0 20px 0;
    font-size: 13px;
    color: var(--content-fg-secondary);
}

.cards {
    display: flex;
    flex-direction: column;
    gap: 16px;
}

.setting-card {
    border-radius: var(--card-radius);
    box-shadow: var(--card-shadow);
    background: var(--card-bg);
    border: 1px solid var(--card-border);
    transition: box-shadow 0.3s ease;
}
.setting-card:hover {
    box-shadow: var(--card-shadow-hover);
}

.card-header {
    display: flex;
    flex-direction: column;
    gap: 2px;
}
.card-title {
    font-size: 15px;
    font-weight: 600;
    color: var(--content-fg);
}
.card-sub {
    font-size: 12px;
    color: var(--content-fg-secondary);
}

.hint {
    margin-left: 8px;
    color: var(--content-fg-secondary);
}
.hint-block {
    margin-top: 6px;
    font-size: 12px;
    color: var(--content-fg-secondary);
}

.backup-actions {
    display: flex;
    gap: 8px;
}

/* 响应式：中等屏宽收窄内容上限，避免宽屏右侧留白过多 */
@media (max-width: 1024px) {
    .page {
        max-width: 100%;
    }
}

/* 响应式：窄屏表单 label 顶部对齐、内边距紧凑 */
@media (max-width: 768px) {
    .page {
        padding: 16px;
    }
    .page-title h2 {
        font-size: 18px;
    }
    /* el-form label 在窄屏改为顶部对齐，避免标签挤压输入框 */
    :deep(.setting-form .el-form-item__label) {
        float: none;
        display: block;
        text-align: left;
        padding: 0 0 6px 0;
        line-height: 1.5;
    }
    :deep(.setting-form .el-form-item__content) {
        margin-left: 0 !important;
    }
}
</style>
