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

  // 绑定 Wails 事件 "log:append"，由后端 EventsEmit 推送。
  function bindEvents(runtime: any) {
    if (bound || !runtime?.EventsOn) return
    runtime.EventsOn('log:append', (line: LogLine) => append(line))
    bound = true
  }

  function clear() {
    lines.value = []
  }

  return { lines, append, bindEvents, clear }
})
