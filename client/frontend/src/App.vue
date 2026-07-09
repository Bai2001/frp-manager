<script lang="ts" setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Events } from '@wailsio/runtime'
import {
    Connection,
    Position,
    Document,
    Setting,
    Fold,
    Expand,
} from '@element-plus/icons-vue'
import { useLogStore } from '@/stores/log'

const route = useRoute()
const router = useRouter()
const active = computed(() => route.path)
const logStore = useLogStore()

// 侧边栏折叠状态：窄屏自动收起，用户也可手动切换
const collapsed = ref(false)

// 菜单项与图标映射
const menus = [
    { index: '/servers', label: '服务器', desc: '管理 frps 服务端', icon: Connection },
    { index: '/tunnels', label: '映射', desc: '配置内网穿透', icon: Position },
    { index: '/logs', label: '日志', desc: '查看运行日志', icon: Document },
    { index: '/settings', label: '设置', desc: '应用偏好设置', icon: Setting },
]

// 监听窗口尺寸：≤900px 自动折叠侧边栏
function handleResize() {
    collapsed.value = window.innerWidth <= 900
}

onMounted(() => {
    handleResize()
    window.addEventListener('resize', handleResize)
    // 绑定后端 app.Event.Emit 的 log:append 事件
    // v3 的 Events.On 回调收到的是 WailsEvent 对象，数据在 ev.data 里
    Events.On('log:append', (ev: any) => {
        const line = ev?.data ?? ev
        logStore.append({
            time: line.time ?? new Date().toISOString(),
            level: line.level ?? 'info',
            message: line.message ?? '',
            server_id: line.server_id,
        })
    })
})

onUnmounted(() => {
    window.removeEventListener('resize', handleResize)
})
</script>

<template>
    <el-container class="layout">
        <!-- 宽度由 CSS 控制，避免 el-aside :width 与 collapse 双重重排导致卡顿 -->
        <el-aside width="220px" class="aside" :class="{ collapsed }">
            <div class="brand">
                <div class="brand-logo">F</div>
                <div class="brand-text">
                    <div class="brand-title">FRP Manager</div>
                    <div class="brand-subtitle">内网穿透管理</div>
                </div>
            </div>
            <!-- 不用 el-menu collapse，文字用 CSS 渐隐，避免菜单结构瞬切 -->
            <el-menu :default-active="active" class="side-menu" @select="(i: string) => router.push(i)">
                <el-menu-item v-for="m in menus" :key="m.index" :index="m.index" class="side-menu-item">
                    <el-icon class="menu-icon"><component :is="m.icon" /></el-icon>
                    <template #title>
                        <span class="menu-label">{{ m.label }}</span>
                    </template>
                </el-menu-item>
            </el-menu>
            <div class="aside-footer">
                <el-button
                    class="collapse-btn"
                    link
                    :icon="collapsed ? Expand : Fold"
                    @click="collapsed = !collapsed"
                >
                    <span class="collapse-text">收起</span>
                </el-button>
                <div class="version">v0.2.0</div>
            </div>
        </el-aside>
        <el-main class="main">
            <router-view />
        </el-main>
    </el-container>
</template>

<style scoped>
.layout {
    height: 100vh;
}

/* 侧边栏：固定过渡宽度，折叠态覆盖为 64px */
.aside {
    width: 220px !important;
    flex-shrink: 0;
    background: var(--sidebar-bg);
    display: flex;
    flex-direction: column;
    border-right: 1px solid var(--sidebar-border);
    overflow: hidden;
    transition: width 0.28s cubic-bezier(0.4, 0, 0.2, 1);
    will-change: width;
}
.aside.collapsed {
    width: 64px !important;
}

/* 品牌区 */
.brand {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 20px 18px;
    border-bottom: 1px solid var(--sidebar-border);
    min-height: 76px;
    box-sizing: border-box;
    transition: padding 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}
.aside.collapsed .brand {
    justify-content: center;
    padding: 20px 14px;
    gap: 0;
}
.brand-logo {
    width: 36px;
    height: 36px;
    border-radius: 10px;
    background: var(--brand-gradient);
    color: #fff;
    font-size: 20px;
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
    flex-shrink: 0;
}
.brand-text {
    display: flex;
    flex-direction: column;
    line-height: 1.3;
    overflow: hidden;
    white-space: nowrap;
    opacity: 1;
    max-width: 140px;
    transition: opacity 0.2s ease, max-width 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}
.aside.collapsed .brand-text {
    opacity: 0;
    max-width: 0;
}
.brand-title {
    font-size: 15px;
    font-weight: 600;
    color: var(--sidebar-fg);
}
.brand-subtitle {
    font-size: 11px;
    color: var(--sidebar-fg-muted);
}

/* 菜单 */
.side-menu {
    flex: 1;
    background: transparent;
    border-right: none;
    padding: 12px 10px;
    overflow: hidden;
    --el-menu-bg-color: transparent;
    --el-menu-text-color: var(--sidebar-fg);
    --el-menu-hover-bg-color: var(--sidebar-hover-bg);
    --el-menu-active-color: var(--sidebar-fg);
    transition: padding 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}
.aside.collapsed .side-menu {
    padding: 12px 8px;
}
.side-menu-item {
    height: 44px;
    line-height: 44px;
    margin-bottom: 4px;
    border-radius: 8px;
    color: var(--sidebar-fg);
    padding-left: 14px !important;
    padding-right: 14px !important;
    transition: background-color 0.2s ease, color 0.2s ease, padding 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}
.aside.collapsed .side-menu-item {
    padding-left: 0 !important;
    padding-right: 0 !important;
    justify-content: center;
}
.side-menu-item:hover {
    background: var(--sidebar-hover-bg);
    color: var(--sidebar-fg);
}
.side-menu-item.is-active {
    background: var(--sidebar-active-bg);
    color: var(--sidebar-fg);
    position: relative;
}
.side-menu-item.is-active::before {
    content: '';
    position: absolute;
    left: 0;
    top: 50%;
    transform: translateY(-50%);
    width: 3px;
    height: 20px;
    border-radius: 2px;
    background: var(--sidebar-active-bar);
}
.menu-icon {
    font-size: 18px;
    margin-right: 10px;
    flex-shrink: 0;
    transition: margin 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}
.aside.collapsed .menu-icon {
    margin-right: 0;
}
.menu-label {
    display: inline-block;
    overflow: hidden;
    white-space: nowrap;
    opacity: 1;
    max-width: 120px;
    vertical-align: middle;
    transition: opacity 0.18s ease, max-width 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}
.aside.collapsed .menu-label {
    opacity: 0;
    max-width: 0;
}

/* 底部 */
.aside-footer {
    padding: 12px 18px;
    border-top: 1px solid var(--sidebar-border);
    display: flex;
    flex-direction: column;
    align-items: stretch;
    transition: padding 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}
.aside.collapsed .aside-footer {
    padding: 12px 8px;
    align-items: center;
}
.collapse-btn {
    color: var(--sidebar-fg-muted);
    width: 100%;
    justify-content: flex-start;
    transition: color 0.2s ease;
}
.aside.collapsed .collapse-btn {
    justify-content: center;
    width: auto;
}
.collapse-btn:hover {
    color: var(--sidebar-fg);
}
.collapse-text {
    overflow: hidden;
    white-space: nowrap;
    opacity: 1;
    max-width: 48px;
    margin-left: 4px;
    transition: opacity 0.18s ease, max-width 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}
.aside.collapsed .collapse-text {
    opacity: 0;
    max-width: 0;
    margin-left: 0;
}
.version {
    font-size: 11px;
    color: var(--sidebar-fg-muted);
    margin-top: 4px;
    overflow: hidden;
    white-space: nowrap;
    opacity: 1;
    max-height: 20px;
    transition: opacity 0.18s ease, max-height 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}
.aside.collapsed .version {
    opacity: 0;
    max-height: 0;
    margin-top: 0;
}

/* 主内容区 */
.main {
    padding: 0;
    background: var(--content-bg);
    /* 滚动由主内容区统一承担，滚动条贴应用右边框；
       各视图 .page 只负责内容留白（padding），不再各自滚动 */
    overflow: auto;
    height: 100%;
    display: flex;
    flex-direction: column;
    min-width: 0;
}
</style>
