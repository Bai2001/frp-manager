import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useServerStore } from '@/stores/server'

export type LogLevel = 'info' | 'warn' | 'error'

export interface LogLine {
    time: string
    level: LogLevel
    message: string
    server_id?: string
}

// 筛选级别
export type LevelFilter = 'all' | LogLevel

export const useLogStore = defineStore('log', () => {
    const lines = ref<LogLine[]>([])
    const keyword = ref('')
    const levelFilter = ref<LevelFilter>('all')
    const serverFilter = ref<string>('') // 空 = 全部服务器

    function append(line: LogLine) {
        lines.value.push(line)
        if (lines.value.length > 2000) {
            lines.value.splice(0, lines.value.length - 2000)
        }
    }

    function clear() {
        lines.value = []
    }

    /**
     * 将 server_id 解析为服务器名称
     */
    function serverName(serverId?: string): string {
        if (!serverId) return '系统'
        const serverStore = useServerStore()
        const s = serverStore.servers.find((x) => x.id === serverId)
        return s?.name ?? serverId.slice(0, 8)
    }

    /**
     * 按关键字 + 级别 + 服务器过滤后的日志
     */
    const filtered = computed(() => {
        const kw = keyword.value.trim().toLowerCase()
        return lines.value.filter((l) => {
            if (levelFilter.value !== 'all' && l.level !== levelFilter.value) return false
            if (serverFilter.value && l.server_id !== serverFilter.value) return false
            if (kw) {
                const hay = (l.message + ' ' + serverName(l.server_id)).toLowerCase()
                if (!hay.includes(kw)) return false
            }
            return true
        })
    })

    /**
     * 按级别统计（基于全量日志，不受筛选影响）
     */
    const counts = computed(() => {
        const c = { info: 0, warn: 0, error: 0 }
        for (const l of lines.value) c[l.level]++
        return c
    })

    /**
     * 涉及的服务器 ID 列表（去重，用于筛选下拉）
     */
    const serverIds = computed(() => {
        const set = new Set<string>()
        for (const l of lines.value) {
            if (l.server_id) set.add(l.server_id)
        }
        return Array.from(set)
    })

    return {
        lines,
        keyword,
        levelFilter,
        serverFilter,
        filtered,
        counts,
        serverIds,
        append,
        clear,
        serverName,
    }
})
