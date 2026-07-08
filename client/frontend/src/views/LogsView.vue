<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { Delete, ArrowDown } from '@element-plus/icons-vue'
import { useLogStore } from '@/stores/log'

const store = useLogStore()
const logContainer = ref<HTMLElement | null>(null)
const autoScroll = ref(true)

// 按级别统计
const counts = computed(() => {
    const c = { info: 0, warn: 0, error: 0 }
    for (const l of store.lines) c[l.level]++
    return c
})

// 日志行格式化时间（去掉年份和毫秒，仅保留时分秒）
function fmtTime(t: string): string {
    if (!t) return ''
    // 兼容 ISO 字符串与已格式化字符串
    const d = new Date(t)
    if (isNaN(d.getTime())) return t.length > 8 ? t.slice(11, 19) : t
    return d.toLocaleTimeString('zh-CN', { hour12: false })
}

// 自动滚动到底部
watch(
    () => store.lines.length,
    async () => {
        if (!autoScroll.value) return
        await nextTick()
        if (logContainer.value) {
            logContainer.value.scrollTop = logContainer.value.scrollHeight
        }
    },
)
</script>

<template>
    <div class="page">
        <div class="page-header">
            <div class="page-title">
                <h2>日志</h2>
                <p class="page-desc">实时运行日志（最近 2000 条）</p>
            </div>
            <div class="actions">
                <div class="stat-badge info">
                    <span class="stat-dot"></span>
                    <span class="stat-label">INFO</span>
                    <span class="stat-count">{{ counts.info }}</span>
                </div>
                <div class="stat-badge warn">
                    <span class="stat-dot"></span>
                    <span class="stat-label">WARN</span>
                    <span class="stat-count">{{ counts.warn }}</span>
                </div>
                <div class="stat-badge error">
                    <span class="stat-dot"></span>
                    <span class="stat-label">ERROR</span>
                    <span class="stat-count">{{ counts.error }}</span>
                </div>
                <el-tooltip content="自动滚动到底部" placement="top">
                    <el-button
                        size="small"
                        :type="autoScroll ? 'primary' : 'default'"
                        :icon="ArrowDown"
                        @click="autoScroll = !autoScroll"
                    >{{ autoScroll ? '自动' : '手动' }}</el-button>
                </el-tooltip>
                <el-button size="small" :icon="Delete" @click="store.clear">清空</el-button>
            </div>
        </div>

        <div class="log-view" ref="logContainer">
            <div v-for="(line, i) in store.lines" :key="i" class="log-line" :class="line.level">
                <span class="log-no">{{ i + 1 }}</span>
                <span class="log-time">{{ fmtTime(line.time) }}</span>
                <span class="log-level">[{{ line.level.toUpperCase() }}]</span>
                <span class="log-msg">{{ line.message }}</span>
            </div>
            <div v-if="store.lines.length === 0" class="log-empty">
                <span>暂无日志</span>
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

/* 统计徽章 */
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
}
.log-line:hover {
    background: rgba(255, 255, 255, 0.03);
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
    margin-right: 12px;
    flex-shrink: 0;
}
.log-level {
    margin-right: 10px;
    flex-shrink: 0;
    font-weight: 600;
}
.log-line.info .log-level { color: #7dcfff; }
.log-line.warn .log-level { color: #e0af68; }
.log-line.error .log-level { color: #f7768e; }
.log-line.error .log-msg { color: #f7768e; }
.log-msg {
    word-break: break-all;
    white-space: pre-wrap;
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
