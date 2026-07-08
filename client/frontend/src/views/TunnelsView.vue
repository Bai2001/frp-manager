<script setup lang="ts">
import { onMounted, onUnmounted, ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, VideoPlay, VideoPause, RefreshRight, Delete, Share, CopyDocument } from '@element-plus/icons-vue'
import { useServerStore } from '@/stores/server'
import { useTunnelStore } from '@/stores/tunnel'
import { api, type AddTunnelInput, type Capabilities, type TunnelInfo, type ServerInfo } from '@/api'
import TunnelFormDialog from '@/components/TunnelFormDialog.vue'

const serverStore = useServerStore()
const tunnelStore = useTunnelStore()

const selectedServerId = ref('')
const dialogVisible = ref(false)
const runningMap = ref<Record<string, boolean>>({})
// 能力缓存：serverId -> Capabilities，用于拼接完整可访问 URL
const capabilitiesMap = ref<Record<string, Capabilities>>({})
let pollTimer: number | undefined

const hasServer = computed(() => serverStore.servers.length > 0)

// 协议对应的标签类型
const protocolTagType: Record<string, string> = {
    tcp: '',
    udp: 'info',
    http: 'warning',
    https: 'danger',
}

onMounted(async () => {
    await serverStore.refresh()
    if (serverStore.servers.length > 0) {
        selectedServerId.value = serverStore.servers[0].id
        await refreshTunnels()
        startPolling()
    }
})

onUnmounted(() => {
    if (pollTimer) window.clearInterval(pollTimer)
})

async function refreshTunnels() {
    if (selectedServerId.value) {
        await tunnelStore.refresh(selectedServerId.value)
        await pollRunning()
        // 静默获取当前服务器能力，用于 URL 拼接（失败不提示）
        await loadCapabilities(selectedServerId.value)
    }
}

// 获取并缓存服务器能力（不影响主流程）
async function loadCapabilities(serverId: string) {
    if (capabilitiesMap.value[serverId]) return
    try {
        capabilitiesMap.value[serverId] = await api.checkServerCapabilities(serverId)
    } catch {
        // 能力获取失败不阻断，URL 列降级显示原始字段
    }
}

function startPolling() {
    if (pollTimer) window.clearInterval(pollTimer)
    pollTimer = window.setInterval(pollRunning, 3000)
}

async function pollRunning() {
    for (const tu of tunnelStore.tunnels) {
        runningMap.value[tu.server_id] = await tunnelStore.isRunning(tu.server_id)
    }
}

function openCreate() {
    if (!selectedServerId.value) {
        ElMessage.warning('请先添加服务器')
        return
    }
    dialogVisible.value = true
}

async function handleSubmit(input: AddTunnelInput) {
    const ok = await tunnelStore.addTunnel(input)
    if (ok) {
        dialogVisible.value = false
        await refreshTunnels()
    }
}

async function handleDelete(id: string) {
    await ElMessageBox.confirm('确认删除该映射？将释放服务端资源。', '确认', { type: 'warning' })
    await tunnelStore.deleteTunnel(id)
    await refreshTunnels()
}

async function handleStart() {
    if (!selectedServerId.value) return
    await tunnelStore.startFrpc(selectedServerId.value)
    await pollRunning()
}

async function handleStop() {
    if (!selectedServerId.value) return
    await tunnelStore.stopFrpc(selectedServerId.value)
    await pollRunning()
}

async function handleRestart() {
    if (!selectedServerId.value) return
    await tunnelStore.restartFrpc(selectedServerId.value)
    await pollRunning()
}

async function onServerChange() {
    await refreshTunnels()
    startPolling()
}

/**
 * 查找当前选中服务器信息
 */
function currentServer(): ServerInfo | undefined {
    return serverStore.servers.find((s) => s.id === selectedServerId.value)
}

/**
 * 拼接映射的完整可访问 URL
 * - TCP/UDP: host:remote_port（裸地址，非 http）
 * - HTTP: http://domain 或 http://host:vhost_http_port（非 80 时带端口）
 * - HTTPS: https://domain 或 https://host:vhost_https_port（非 443 时带端口）
 * - subdomain 模式: 前缀 + subdomain_host 拼接
 */
function buildAccessUrl(tu: TunnelInfo): string {
    const caps = capabilitiesMap.value[tu.server_id]
    const server = currentServer()
    const host = server?.host ?? ''

    // TCP / UDP：仅展示公网地址 + 远程端口
    if (tu.protocol === 'tcp' || tu.protocol === 'udp') {
        if (!tu.remote_port) return ''
        return `${host}:${tu.remote_port}`
    }

    // HTTP / HTTPS：拼接域名
    if (tu.protocol === 'http' || tu.protocol === 'https') {
        let domain = tu.custom_domain
        // 子域名模式：前缀 + subdomain_host
        if (!domain && tu.subdomain && caps?.subdomain_host) {
            domain = `${tu.subdomain}.${caps.subdomain_host}`
        }
        // 回退：直接用服务器 host + vhost 端口
        if (!domain) {
            const vhostPort = tu.protocol === 'https' ? caps?.vhost_https_port : caps?.vhost_http_port
            if (!vhostPort) return ''
            const portSuffix = (tu.protocol === 'http' && vhostPort === 80) || (tu.protocol === 'https' && vhostPort === 443) ? '' : `:${vhostPort}`
            return `${tu.protocol}://${host}${portSuffix}`
        }
        // 自定义域名 / 子域名：默认 80/443，无需带 vhost 端口
        const scheme = tu.protocol === 'https' ? 'https' : 'http'
        return `${scheme}://${domain}`
    }

    return ''
}

/**
 * 复制 URL 到剪贴板
 */
async function copyUrl(url: string) {
    if (!url) {
        ElMessage.warning('暂无可复制的地址')
        return
    }
    try {
        await navigator.clipboard.writeText(url)
        ElMessage.success('已复制: ' + url)
    } catch (e: any) {
        ElMessage.error('复制失败: ' + (e?.message ?? e))
    }
}
</script>

<template>
    <div class="page">
        <div class="page-header">
            <div class="page-title">
                <h2>映射</h2>
                <p class="page-desc">配置内网穿透规则</p>
            </div>
            <div class="actions">
                <el-select v-model="selectedServerId" placeholder="选择服务器" @change="onServerChange" style="width: 180px">
                    <el-option v-for="s in serverStore.servers" :key="s.id" :label="s.name" :value="s.id" />
                </el-select>
                <el-button type="primary" :icon="Plus" @click="openCreate" :disabled="!hasServer">创建映射</el-button>
                <el-button :icon="VideoPlay" @click="handleStart" :disabled="!hasServer">启动</el-button>
                <el-button :icon="VideoPause" @click="handleStop" :disabled="!hasServer">停止</el-button>
                <el-button :icon="RefreshRight" @click="handleRestart" :disabled="!hasServer">重启</el-button>
            </div>
        </div>

        <el-card class="table-card" shadow="never" v-loading="tunnelStore.loading">
            <el-table :data="tunnelStore.tunnels" stripe empty-text="暂无映射" class="modern-table">
                <el-table-column prop="name" label="名称" min-width="140" />
                <el-table-column label="协议" width="100">
                    <template #default="{ row }">
                        <el-tag :type="protocolTagType[row.protocol]" size="small" effect="light">
                            {{ row.protocol.toUpperCase() }}
                        </el-tag>
                    </template>
                </el-table-column>
                <el-table-column label="本地" min-width="160">
                    <template #default="{ row }">
                        <span class="mono">{{ row.local_ip }}:{{ row.local_port }}</span>
                    </template>
                </el-table-column>
                <el-table-column label="远程" min-width="240">
                    <template #default="{ row }">
                        <div class="remote-cell">
                            <span class="mono">{{ buildAccessUrl(row) || '—' }}</span>
                            <el-button
                                v-if="buildAccessUrl(row)"
                                size="small"
                                link
                                :icon="CopyDocument"
                                @click="copyUrl(buildAccessUrl(row))"
                                class="copy-btn"
                            />
                        </div>
                    </template>
                </el-table-column>
                <el-table-column label="frpc 状态" width="130">
                    <template #default="{ row }">
                        <div class="status-cell">
                            <span class="status-dot" :class="runningMap[row.server_id] ? 'running' : 'stopped'"></span>
                            <el-tag :type="runningMap[row.server_id] ? 'success' : 'info'" size="small" effect="light">
                                {{ runningMap[row.server_id] ? '运行中' : '已停止' }}
                            </el-tag>
                        </div>
                    </template>
                </el-table-column>
                <el-table-column label="操作" width="120" fixed="right">
                    <template #default="{ row }">
                        <el-button size="small" link type="danger" :icon="Delete" @click="handleDelete(row.id)">删除</el-button>
                    </template>
                </el-table-column>
                <template #empty>
                    <div class="empty-state">
                        <el-icon class="empty-icon"><Share /></el-icon>
                        <p class="empty-text">还没有映射，选择服务器后创建</p>
                    </div>
                </template>
            </el-table>
        </el-card>

        <TunnelFormDialog v-model:visible="dialogVisible" :server-id="selectedServerId" @submit="handleSubmit" />
    </div>
</template>

<style scoped>
.page {
    padding: 24px;
    flex: 1;
    min-height: 0;
}

.page-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 20px;
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
    flex-wrap: wrap;
}

/* 卡片化表格 */
.table-card {
    border-radius: var(--card-radius);
    box-shadow: var(--card-shadow);
    border: none;
    transition: box-shadow 0.3s ease;
}
.table-card:hover {
    box-shadow: var(--card-shadow-hover);
}

:deep(.modern-table) {
    --el-table-border-color: transparent;
    --el-table-header-bg-color: #fafbfc;
}
:deep(.modern-table th.el-table__cell) {
    font-weight: 600;
    color: var(--content-fg);
}
:deep(.modern-table .el-table__row:hover > td) {
    background: #f0f5ff !important;
}

.mono {
    font-family: 'JetBrains Mono', Consolas, 'Courier New', monospace;
    font-size: 13px;
}

/* 远程单元格：URL + 复制按钮 */
.remote-cell {
    display: flex;
    align-items: center;
    gap: 4px;
}
.remote-cell .mono {
    color: var(--brand-color);
    word-break: break-all;
}
.copy-btn {
    color: var(--content-fg-secondary);
    padding: 2px;
}
.copy-btn:hover {
    color: var(--brand-color);
}

/* 状态单元格 */
.status-cell {
    display: flex;
    align-items: center;
    gap: 8px;
}
.status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
}
.status-dot.running {
    background: var(--success);
    box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.2);
    animation: pulse 1.5s ease-in-out infinite;
}
.status-dot.stopped {
    background: #d1d5db;
}
@keyframes pulse {
    0%, 100% { box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.2); }
    50% { box-shadow: 0 0 0 6px rgba(16, 185, 129, 0.1); }
}

/* 空状态 */
.empty-state {
    padding: 48px 0;
    color: var(--content-fg-secondary);
}
.empty-icon {
    font-size: 48px;
    color: #d3d6db;
    margin-bottom: 12px;
}
.empty-text {
    margin: 8px 0 0 0;
    font-size: 13px;
}

/* 响应式：窄屏紧凑化 */
@media (max-width: 768px) {
    .page {
        padding: 16px;
    }
    .page-header {
        flex-direction: column;
        align-items: stretch;
        gap: 12px;
    }
    .page-title h2 {
        font-size: 18px;
    }
    .actions {
        gap: 6px;
    }
    .actions .el-button {
        margin-left: 0 !important;
    }
}
</style>
