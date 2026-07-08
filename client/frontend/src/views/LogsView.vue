<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Delete, ArrowDown, Search, Download, CopyDocument } from '@element-plus/icons-vue'
import { useLogStore } from '@/stores/log'
import { useServerStore } from '@/stores/server'

const store = useLogStore()
const serverStore = useServerStore()
const logContainer = ref<HTMLElement | null>(null)
const autoScroll = ref(true)

// 进入日志页时确保服务器列表已加载，用于 server_id -> 名称解析
onMounted(() => {
    if (serverStore.servers.length === 0) {
        serverStore.refresh()
    }
})

// 日志行格式化时间（去掉年份和毫秒，仅保留时分秒）
function fmtTime(t: string): string {
    if (!t) return ''
    const d = new Date(t)
    if (isNaN(d.getTime())) return t.length > 8 ? t.slice(11, 19) : t
    return d.toLocaleTimeString('zh-CN', { hour12: false })
}

// 自动滚动到底部（基于筛选后日志行数变化）
watch(
    () => store.filtered.length,
    async () => {
        if (!autoScroll.value) return
        await nextTick()
        if (logContainer.value) {
            logContainer.value.scrollTop = logContainer.value.scrollHeight
        }
    },
)

/**
 * 复制单条日志
 */
async function copyLine(line: any) {
    const text = `[${fmtTime(line.time)}] [${line.level.toUpperCase()}] [${store.serverName(line.server_id)}] ${line.message}`
    try {
        await navigator.clipboard.writeText(text)
        ElMessage.success('已复制该行日志')
    } catch (e: any) {
        ElMessage.error('复制失败: ' + (e?.message ?? e))
    }
}

/**
 * 导出全部（筛选后）日志为 txt 文件
 */
function exportLogs() {
    if (store.filtered.length === 0) {
        ElMessage.warning('暂无可导出的日志')
        return
    }
    const content = store.filtered
        .map((l) => `[${l.time}] [${l.level.toUpperCase()}] [${store.serverName(l.server_id)}] ${l.message}`)
        .join('\n')
    const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    const ts = new Date().toISOString().slice(0, 10)
    a.download = `frp-logs-${ts}.txt`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    ElMessage.success(`已导出 ${store.filtered.length} 条日志`)
}

/**
 * 级别徽章点击快速筛选
 */
function toggleLevel(level: 'info' | 'warn' | 'error') {
    store.levelFilter = store.levelFilter === level ? 'all' : level
}
</script>

<template>
    <div class="page">
        <div class="page-header">
            <div class="page-title">
                <h2>日志</h2>
                <p class="page-desc">实时运行日志（最近 2000 条）</p>
            </div>
            <div class="actions">
                <div class="stat-badge info" :class="{ active: store.levelFilter === 'info' }" @click="toggleLevel('info')">
                    <span class="stat-dot"></span>
                    <span class="stat-label">INFO</span>
                    <span class="stat-count">{{ store.counts.info }}</span>
                </div>
                <div class="stat-badge warn" :class="{ active: store.levelFilter === 'warn' }" @click="toggleLevel('warn')">
                    <span class="stat-dot"></span>
                    <span class="stat-label">WARN</span>
                    <span class="stat-count">{{ store.counts.warn }}</span>
                </div>
                <div class="stat-badge error" :class="{ active: store.levelFilter === 'error' }" @click="toggleLevel('error')">
                    <span class="stat-dot"></span>
                    <span class="stat-label">ERROR</span>
                    <span class="stat-count">{{ store.counts.error }}</span>
                </div>
                <el-tooltip content="自动滚动到底部" placement="top">
                    <el-button
                        size="small"
                        :type="autoScroll ? 'primary' : 'default'"
                        :icon="ArrowDown"
                        @click="autoScroll = !autoScroll"
                    >{{ autoScroll ? '自动' : '手动' }}</el-button>
                </el-tooltip>
                <el-button size="small" :icon="Download" @click="exportLogs">导出</el-button>
                <el-button size="small" :icon="Delete" @click="store.clear">清空</el-button>
            </div>
        </div>

        <!-- 筛选工具栏 -->
        <div class="filter-bar">
            <el-input
                v-model="store.keyword"
                placeholder="搜索日志内容..."
                :prefix-icon="Search"
                clearable
                size="small"
                class="search-input"
            />
            <el-select v-model="store.serverFilter" placeholder="全部服务器" clearable size="small" class="server-select">
                <el-option label="全部服务器" value="" />
                <el-option
                    v-for="sid in store.serverIds"
                    :key="sid"
                    :label="store.serverName(sid)"
                    :value="sid"
                />
            </el-select>
            <span class="filter-summary">
                共 {{ store.lines.length }} 条，筛选后 {{ store.filtered.length }} 条
            </span>
        </div>

        <!-- 终端日志区 -->
        <div class="log-view" ref="logContainer">
            <div
                v-for="(line, i) in store.filtered"
                :key="i"
                class="log-line"
                :class="line.level"
            >
                <span class="log-no">{{ i + 1 }}</span>
                <span class="log-time">{{ fmtTime(line.time) }}</span>
                <span class="log-level">[{{ line.level.toUpperCase() }}]</span>
                <span class="log-source" :title="line.server_id">{{ store.serverName(line.server_id) }}</span>
                <span class="log-msg">{{ line.message }}</span>
                <el-button
                    class="copy-row"
                    size="small"
                    link
                    :icon="CopyDocument"
                    @click="copyLine(line)"
                />
            </div>
            <div v-if="store.filtered.length === 0" class="log-empty">
                <span>{{ store.lines.length === 0 ? '暂无日志' : '没有匹配的日志' }}</span>
            </div>
        </div>
    </div>
</template>

<style scoped>
.page {
    padding: 24px;
    height: 100%;
    display: flex;
    flex-direction: column;
}

.page-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 16px;
    flex-wrap: wrap;
    gap: 12px;
}
.page-title h2 {
    margin: 0 0 4px 0;
    font-size: 20px;
    font-weight: 600;
    color: var(--content-fg);
}
.page-desc {
    margin: 0;
    font-size: 13px;
    color: var(--content-fg-secondary);
}
.actions {
    display: flex;
    gap: 8px;
    align-items: center;
    flex-wrap: wrap;
}

/* 筛选工具栏 */
.filter-bar {
    display: flex;
    gap: 10px;
    align-items: center;
    margin-bottom: 12px;
    flex-wrap: wrap;
}
.search-input {
    width: 240px;
}
.server-select {
    width: 160px;
}
.filter-summary {
    font-size: 12px;
    color: var(--content-fg-secondary);
    margin-left: auto;
}

/* 统计徽章（可点击筛选） */
.stat-badge {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 4px 10px;
    border-radius: 16px;
    font-size: 12px;
    font-weight: 500;
    background: #fff;
    border: 1px solid #ebeef5;
    cursor: pointer;
    user-select: none;
    transition: all 0.2s ease;
}
.stat-badge:hover {
    border-color: var(--brand-color);
}
.stat-badge.active {
    background: var(--brand-color);
    border-color: var(--brand-color);
}
.stat-badge.active .stat-label,
.stat-badge.active .stat-count {
    color: #fff;
}
.stat-badge .stat-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
}
.stat-badge.info .stat-dot { background: var(--info); }
.stat-badge.warn .stat-dot { background: var(--warning); }
.stat-badge.error .stat-dot { background: var(--danger); }
.stat-badge .stat-label {
    color: var(--content-fg-secondary);
    letter-spacing: 0.5px;
}
.stat-badge .stat-count {
    font-weight: 600;
    color: var(--content-fg);
}

/* 终端 */
.log-view {
    flex: 1;
    overflow: auto;
    background: linear-gradient(180deg, var(--terminal-bg) 0%, var(--terminal-bg-soft) 100%);
    border-radius: var(--card-radius);
    padding: 14px 16px;
    font-family: 'JetBrains Mono', Consolas, 'Courier New', monospace;
    font-size: 12.5px;
    line-height: 1.7;
    box-shadow: var(--card-shadow);
}
.log-line {
    display: flex;
    align-items: baseline;
    color: var(--terminal-fg);
    padding: 1px 0;
    border-radius: 3px;
    position: relative;
}
.log-line:hover {
    background: rgba(255, 255, 255, 0.03);
}
.log-line:hover .copy-row {
    opacity: 1;
}
.log-no {
    color: #565f89;
    width: 44px;
    flex-shrink: 0;
    user-select: none;
    text-align: right;
    padding-right: 12px;
}
.log-time {
    color: #7aa2f7;
    margin-right: 10px;
    flex-shrink: 0;
}
.log-level {
    margin-right: 8px;
    flex-shrink: 0;
    font-weight: 600;
}
.log-line.info .log-level { color: #7dcfff; }
.log-line.warn .log-level { color: #e0af68; }
.log-line.error .log-level { color: #f7768e; }
.log-line.error .log-msg { color: #f7768e; }
.log-source {
    color: #9ece6a;
    margin-right: 10px;
    flex-shrink: 0;
    max-width: 120px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}
.log-msg {
    word-break: break-all;
    white-space: pre-wrap;
    flex: 1;
    min-width: 0;
}
/* 单行复制按钮：默认隐藏，悬浮显示 */
.copy-row {
    opacity: 0;
    color: #565f89;
    flex-shrink: 0;
    margin-left: 4px;
    transition: opacity 0.2s ease;
}
.copy-row:hover {
    color: #7aa2f7;
}

/* 空状态 */
.log-empty {
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #565f89;
    font-size: 13px;
}

/* 终端滚动条 */
.log-view::-webkit-scrollbar {
    width: 6px;
}
.log-view::-webkit-scrollbar-track {
    background: transparent;
}
.log-view::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.12);
    border-radius: 3px;
}
.log-view::-webkit-scrollbar-thumb:hover {
    background: rgba(255, 255, 255, 0.2);
}
</style>
