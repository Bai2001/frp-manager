import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
    { path: '/', redirect: '/servers' },
    {
        path: '/servers',
        name: 'servers',
        component: () => import('@/views/ServersView.vue'),
        meta: { title: '服务器' },
    },
    {
        path: '/tunnels',
        name: 'tunnels',
        component: () => import('@/views/TunnelsView.vue'),
        meta: { title: '映射' },
    },
    {
        path: '/logs',
        name: 'logs',
        component: () => import('@/views/LogsView.vue'),
        meta: { title: '日志' },
    },
    {
        path: '/settings',
        name: 'settings',
        component: () => import('@/views/SettingsView.vue'),
        meta: { title: '设置' },
    },
]

const router = createRouter({
    history: createWebHashHistory(),
    routes,
})

export default router
