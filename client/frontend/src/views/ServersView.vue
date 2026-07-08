<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Cpu, Edit, Delete, Coin } from '@element-plus/icons-vue'
import { useServerStore } from '@/stores/server'
import { type AddServerInput, type ServerInfo } from '@/api'
import ServerFormDialog from '@/components/ServerFormDialog.vue'

const store = useServerStore()
const dialogVisible = ref(false)
const editing = ref<ServerInfo | null>(null)

onMounted(() => {
    store.refresh()
})

function openAdd() {
    editing.value = null
    dialogVisible.value = true
}

function openEdit(row: ServerInfo) {
    editing.value = row
    dialogVisible.value = true
}

async function handleSubmit(input: AddServerInput) {
    let ok: boolean
    if (editing.value) {
        ok = await store.updateServer(editing.value.id, input)
    } else {
        ok = await store.addServer(input)
    }
    if (ok) dialogVisible.value = false
}

async function handleDelete(row: ServerInfo) {
    await ElMessageBox.confirm(`确认删除服务器「${row.name}」？其下映射将一并删除。`, '确认', { type: 'warning' })
    await store.deleteServer(row.id)
}

async function handleCheckCapabilities(row: ServerInfo) {
    await store.checkCapabilities(row.id)
}
</script>

<template>
    <div class="page">
        <div class="page-header">
            <div class="page-title">
                <h2>服务器</h2>
                <p class="page-desc">管理你的 frps 服务端节点</p>
            </div>
            <el-button type="primary" :icon="Plus" @click="openAdd">添加服务器</el-button>
        </div>

        <el-card class="table-card" shadow="never" v-loading="store.loading">
            <el-table :data="store.servers" stripe empty-text="暂无服务器" class="modern-table">
                <el-table-column prop="name" label="名称" min-width="140">
                    <template #default="{ row }">
                        <div class="cell-name">
                            <span class="name-text">{{ row.name }}</span>
                            <el-tag v-if="row.is_default" type="warning" size="small" effect="dark" class="default-tag">默认</el-tag>
                        </div>
                    </template>
                </el-table-column>
                <el-table-column prop="host" label="公网地址" min-width="160" />
                <el-table-column prop="frps_port" label="frps 端口" width="110" />
                <el-table-column prop="agent_url" label="Agent 地址" min-width="180" />
                <el-table-column label="操作" width="280" fixed="right">
                    <template #default="{ row }">
                        <el-button size="small" link :icon="Cpu" @click="handleCheckCapabilities(row)">检测能力</el-button>
                        <el-button size="small" link :icon="Edit" @click="openEdit(row)">编辑</el-button>
                        <el-button size="small" link type="danger" :icon="Delete" @click="handleDelete(row)">删除</el-button>
                    </template>
                </el-table-column>
                <template #empty>
                    <div class="empty-state">
                        <el-icon class="empty-icon"><Coin /></el-icon>
                        <p class="empty-text">还没有服务器，点击右上角添加</p>
                    </div>
                </template>
            </el-table>
        </el-card>

        <ServerFormDialog v-model:visible="dialogVisible" :editing="editing" @submit="handleSubmit" />

        <el-dialog v-model="store.capabilities" title="服务端能力" width="560px" v-if="store.capabilities">
            <el-descriptions :column="2" border>
                <el-descriptions-item label="frps 运行">{{ store.capabilities.frps_running ? '是' : '否' }}</el-descriptions-item>
                <el-descriptions-item label="frps 版本">{{ store.capabilities.frps_version || '-' }}</el-descriptions-item>
                <el-descriptions-item label="bind 端口">{{ store.capabilities.bind_port }}</el-descriptions-item>
                <el-descriptions-item label="HTTP vhost">{{ store.capabilities.vhost_http_port || '-' }}</el-descriptions-item>
                <el-descriptions-item label="HTTPS vhost">{{ store.capabilities.vhost_https_port || '-' }}</el-descriptions-item>
                <el-descriptions-item label="子域名根">{{ store.capabilities.subdomain_host || '-' }}</el-descriptions-item>
                <el-descriptions-item label="支持协议">
                    <el-tag v-if="store.capabilities.support_tcp" size="small">TCP</el-tag>
                    <el-tag v-if="store.capabilities.support_udp" size="small">UDP</el-tag>
                    <el-tag v-if="store.capabilities.support_http" size="small">HTTP</el-tag>
                    <el-tag v-if="store.capabilities.support_https" size="small">HTTPS</el-tag>
                </el-descriptions-item>
                <el-descriptions-item label="允许根域名">{{ store.capabilities.allowed_root_domains.join(', ') }}</el-descriptions-item>
            </el-descriptions>
        </el-dialog>
    </div>
</template>

<style scoped>
.page {
    padding: 24px;
}

.page-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 20px;
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

/* 现代表格 */
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

/* 名称单元格 */
.cell-name {
    display: flex;
    align-items: center;
    gap: 8px;
}
.name-text {
    font-weight: 500;
}
.default-tag {
    background: linear-gradient(135deg, #fbbf24, #f59e0b);
    border: none;
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
</style>
