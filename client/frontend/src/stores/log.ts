import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface LogLine {
  time: string
  level: 'info' | 'warn' | 'error'
  message: string
  server_id?: string
}

export const useLogStore = defineStore('log', () => {
  const lines = ref<LogLine[]>([])
  let bound = false

  function append(line: LogLine) {
    lines.value.push(line)
    if (lines.value.length > 2000) {
      lines.value.splice(0, lines.value.length - 2000)
    }
  }

  // 绑定 Wails v3 事件 "log:append"，由后端 app.Event.Emit 推送。
  // App.vue 已通过 Events.On 绑定并调用 append，此处保留接口供兼容。
  function bindEvents(runtime: any) {
    if (bound) return
    bound = true
  }

  function clear() {
    lines.value = []
  }

  return { lines, append, bindEvents, clear }
})
