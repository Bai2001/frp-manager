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

  function append(line: LogLine) {
    lines.value.push(line)
    if (lines.value.length > 2000) {
      lines.value.splice(0, lines.value.length - 2000)
    }
  }

  function clear() {
    lines.value = []
  }

  return { lines, append, clear }
})
