import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import App from './App.vue'
import router from './router'
import './style.css'
import { useSettingsStore } from '@/stores/settings'
import { watchTheme } from '@/composables/useTheme'

const app = createApp(App)
const pinia = createPinia()
app.use(pinia)
app.use(router)
app.use(ElementPlus)
app.mount('#app')

// 启动后加载设置并应用主题（避免阻塞 mount）
const settingsStore = useSettingsStore()
settingsStore
    .load()
    .then(() => {
        watchTheme(settingsStore.settings.theme_mode)
    })
    .catch(() => {
        watchTheme('system')
    })
