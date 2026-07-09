import type { ThemeMode } from '@/api'
import { api } from '@/api'

const MEDIA = '(prefers-color-scheme: dark)'

/**
 * 将 theme_mode 解析为最终 light/dark
 */
export function resolveTheme(mode: ThemeMode | string | undefined): 'light' | 'dark' {
    const m = mode === 'light' || mode === 'dark' || mode === 'system' ? mode : 'system'
    if (m === 'light' || m === 'dark') return m
    if (typeof window !== 'undefined' && window.matchMedia) {
        return window.matchMedia(MEDIA).matches ? 'dark' : 'light'
    }
    return 'light'
}

/**
 * 同步原生标题栏（失败不影响前端主题）
 */
function syncNativeTheme(mode: ThemeMode | string | undefined) {
    const m = mode === 'light' || mode === 'dark' || mode === 'system' ? mode : 'system'
    void api.setNativeTheme(m).catch(() => {
        // 绑定未就绪或非桌面环境时忽略
    })
}

/**
 * 给 html 根节点切换 dark class（Element Plus 暗色依赖 html.dark）
 */
export function applyTheme(mode: ThemeMode | string | undefined) {
    const resolved = resolveTheme(mode)
    document.documentElement.classList.toggle('dark', resolved === 'dark')
    syncNativeTheme(mode)
    return resolved
}

let mediaHandler: ((e: MediaQueryListEvent) => void) | null = null
let mediaList: MediaQueryList | null = null

/**
 * 按 theme_mode 应用主题；system 时监听系统变化
 */
export function watchTheme(mode: ThemeMode | string | undefined) {
    stopWatchTheme()
    applyTheme(mode)
    const m = mode === 'light' || mode === 'dark' || mode === 'system' ? mode : 'system'
    if (m !== 'system' || typeof window === 'undefined' || !window.matchMedia) return
    mediaList = window.matchMedia(MEDIA)
    mediaHandler = () => applyTheme('system')
    mediaList.addEventListener('change', mediaHandler)
}

/**
 * 移除系统主题监听
 */
export function stopWatchTheme() {
    if (mediaList && mediaHandler) {
        mediaList.removeEventListener('change', mediaHandler)
    }
    mediaList = null
    mediaHandler = null
}
