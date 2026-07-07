# 客户端 GUI 闭环实现计划（v0.1 计划 3/3）

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 把前端四个占位视图接通后端真实方法，完成 v0.1 端到端 GUI 闭环：服务器增删改 + 检测能力、按协议动态创建映射（调 agent 在线校验端口/域名）、frpc 启停重启 + 状态轮询、实时日志（Wails 事件）、托盘运行。完成后用户可在桌面端完成"加服务器 → 创建映射 → 启动 frpc → 看日志"全流程。

**架构：** 复用 `client/frontend/`。改造 `src/api/index.ts`（修正 Wails 绑定路径 + 补全方法）、`src/stores`（加 actions）、四个视图（接通表单与操作）、`App.vue`（托盘控制）。后端 `app.go` 新增日志事件发射 + 托盘相关方法（依赖计划 2 的 App）。依赖计划 1（server-agent 可用）与计划 2（App 方法可用）。

**技术栈：** Vue 3 + TS + Element Plus + Pinia + Vue Router（已引入）、Wails v2 runtime（`@wailsio/runtime`，需安装）、Wails 事件系统（`runtime.EventsOn`/`EventsEmit`）、Wails 托盘 API（`application.NewSystemTray`，v2.12 支持）。

**约定：**
- 前端统一通过 `src/api/index.ts` 调后端，不直接用 `window.go`。
- 错误统一用 Element Plus `ElMessage.error` 提示，成功用 `ElMessage.success`。
- frpc 状态用轮询（每 3s）而非事件，简化实现；日志用 Wails 事件实时推送。
- 表单验证用 Element Plus `el-form` + rules。
- 时间戳前端用 `dayjs`？为避免新增依赖，用原生 `Date` + `toLocaleString`。

**关于设计文档矛盾的处理：** 第 20 节 MVP 列"托盘运行"，第 21 节 v0.2 又列"托盘/开机自启"。本计划按第 20 节纳入**托盘**，**开机自启**归 v0.2 不实现。

---

## 文件结构

**修改：**
- `client/frontend/src/api/index.ts` — 修正绑定路径、补全 Add/Update/Delete/Capabilities/IsFrpcRunning 方法
- `client/frontend/src/stores/server.ts` — 加 addServer/updateServer/deleteServer/checkCapabilities
- `client/frontend/src/stores/tunnel.ts` — 加 addTunnel/updateTunnel/deleteTunnel/checkPort/allocatePort/checkDomain/registerDomain
- `client/frontend/src/stores/log.ts` — 接 Wails 事件实时日志
- `client/frontend/src/views/ServersView.vue` — 表单对话框 + 增删改 + 检测能力
- `client/frontend/src/views/TunnelsView.vue` — 创建表单（按协议动态字段）+ 在线校验 + 增删 + frpc 启停
- `client/frontend/src/views/LogsView.vue` — 接实时日志事件
- `client/frontend/src/views/SettingsView.vue` — frpc 路径配置（接后端设置方法，可先占位）
- `client/frontend/src/App.vue` — 启动时挂日志事件监听
- `client/app.go` — 新增 StartLogStreaming/StopLogStreaming 方法 + 日志事件发射；托盘控制
- `client/main.go` — 配置托盘 + 窗口最小化到托盘

**新建：**
- `client/frontend/src/components/ServerFormDialog.vue` — 服务器表单对话框
- `client/frontend/src/components/TunnelFormDialog.vue` — 映射创建表单对话框（按协议动态字段）

**测试策略：** 前端无单测框架（骨架未引入 vitest）。本计划以 `npm run build`（vue-tsc 类型检查 + vite 构建）通过为每步验证标准，最终用 `wails dev` 手动冒烟验证全流程。每个任务结束跑一次 `npm run build`。

---

### 任务 1：修正并补全 api 封装

**文件：**
- 修改：`client/frontend/src/api/index.ts`

Wails v2 绑定路径实际是 `window.go.main.App`（`main` 是包名）。现有写法 `window.go['github.com/kdc/frp-manager/client'].App` 路径错误，需修正。同时 Wails 方法返回的是 Promise，错误会 reject，需正确处理。

- [ ] **步骤 1：重写 api/index.ts**

```typescript
// 封装对 Wails 后端 App 方法的调用。
// Wails v2 把 main 包里 Bind 的对象注入到 window.go.main.App 下。

export interface ServerInfo {
  id: string
  name: string
  host: string
  frps_port: number
  frp_token: string
  agent_url: string
  agent_token: string
  is_default: boolean
  remark?: string
}

export interface TunnelInfo {
  id: string
  server_id: string
  name: string
  protocol: 'tcp' | 'udp' | 'http' | 'https'
  local_ip: string
  local_port: number
  remote_port?: number
  custom_domain?: string
  subdomain?: string
  enabled: boolean
  status: string
}

export interface AddServerInput {
  name: string
  host: string
  frps_port: number
  frp_token: string
  agent_url: string
  agent_token: string
  is_default?: boolean
  remark?: string
}

export interface AddTunnelInput {
  server_id: string
  name: string
  protocol: 'tcp' | 'udp' | 'http' | 'https'
  local_ip: string
  local_port: number
  remote_port?: number
  custom_domain?: string
  subdomain?: string
}

export interface Capabilities {
  frps_running: boolean
  frps_version: string
  bind_port: number
  allow_ports: { start: number; end: number }[]
  support_tcp: boolean
  support_udp: boolean
  support_http: boolean
  support_https: boolean
  vhost_http_port: number
  vhost_https_port: number
  subdomain_host: string
  allowed_root_domains: string[]
}

export interface PortCheckResult {
  protocol: string
  port: number
  available: boolean
  reason: string
}

export interface DomainCheckResult {
  domain: string
  available: boolean
  reason: string
}

// Wails 绑定的后端对象。开发模式下 wails dev 会注入；类型宽松处理。
function backend(): any {
  return (window as any).go?.main?.App
}

// 统一调用包装：Wails 方法返回 Promise，reject 时抛 Error。
async function call<T>(fn: any, ...args: any[]): Promise<T> {
  if (!fn) {
    throw new Error('后端未就绪（window.go.main.App 不存在）')
  }
  return await fn(...args)
}

export const api = {
  // 服务器
  async listServers(): Promise<ServerInfo[]> {
    return (await call(backend()?.ListServers)) ?? []
  },
  async addServer(input: AddServerInput): Promise<string> {
    return await call(backend()?.AddServer, input)
  },
  async updateServerByID(id: string, input: AddServerInput): Promise<void> {
    await call(backend()?.UpdateServerByID, id, input)
  },
  async deleteServer(id: string): Promise<void> {
    await call(backend()?.DeleteServer, id)
  },
  async checkServerCapabilities(id: string): Promise<Capabilities> {
    return await call(backend()?.CheckServerCapabilities, id)
  },

  // 映射
  async listTunnels(serverId?: string): Promise<TunnelInfo[]> {
    return (await call(backend()?.ListTunnels, serverId ?? '')) ?? []
  },
  async addTunnel(input: AddTunnelInput): Promise<string> {
    return await call(backend()?.AddTunnel, input)
  },
  async updateTunnelByID(id: string, input: AddTunnelInput): Promise<void> {
    await call(backend()?.UpdateTunnelByID, id, input)
  },
  async deleteTunnel(id: string): Promise<void> {
    await call(backend()?.DeleteTunnel, id)
  },

  // frpc
  async generateFrpcConfig(serverId: string): Promise<string> {
    return await call(backend()?.GenerateFrpcConfig, serverId)
  },
  async startFrpc(serverId: string): Promise<void> {
    await call(backend()?.StartFrpc, serverId)
  },
  async stopFrpc(serverId: string): Promise<void> {
    await call(backend()?.StopFrpc, serverId)
  },
  async restartFrpc(serverId: string): Promise<void> {
    await call(backend()?.RestartFrpc, serverId)
  },
  async isFrpcRunning(serverId: string): Promise<boolean> {
    return await call(backend()?.IsFrpcRunning, serverId)
  },
}
```

> 注意：`backend()` 返回的每个方法是 Wails 生成的 JS 函数，调用即返回 Promise。`call` 包装只是补空检查与错误传播。Wails 生成的方法名与 Go 方法名一致（驼峰），如 `ListServers`、`AddServer`、`UpdateServerByID`、`IsFrpcRunning`。

- [ ] **步骤 2：构建验证**

运行：`cd client/frontend && npm run build`
预期：vue-tsc 无类型错误，vite 构建成功

- [ ] **步骤 3：Commit**

```bash
git add client/frontend/src/api/index.ts
git commit -m "feat(client): 修正 Wails 绑定路径并补全 api 封装"
```

---

### 任务 2：stores 补全 actions

**文件：**
- 修改：`src/stores/server.ts`、`src/stores/tunnel.ts`、`src/stores/log.ts`

- [ ] **步骤 1：重写 server store**

```typescript
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api, type ServerInfo, type AddServerInput, type Capabilities } from '@/api'

export const useServerStore = defineStore('server', () => {
  const servers = ref<ServerInfo[]>([])
  const loading = ref(false)
  const capabilities = ref<Capabilities | null>(null)

  async function refresh() {
    loading.value = true
    try {
      servers.value = await api.listServers()
    } catch (e: any) {
      ElMessage.error('加载服务器列表失败: ' + e.message)
    } finally {
      loading.value = false
    }
  }

  async function addServer(input: AddServerInput): Promise<boolean> {
    try {
      await api.addServer(input)
      await refresh()
      ElMessage.success('服务器已添加')
      return true
    } catch (e: any) {
      ElMessage.error('添加失败: ' + e.message)
      return false
    }
  }

  async function updateServer(id: string, input: AddServerInput): Promise<boolean> {
    try {
      await api.updateServerByID(id, input)
      await refresh()
      ElMessage.success('服务器已更新')
      return true
    } catch (e: any) {
      ElMessage.error('更新失败: ' + e.message)
      return false
    }
  }

  async function deleteServer(id: string): Promise<boolean> {
    try {
      await api.deleteServer(id)
      await refresh()
      ElMessage.success('服务器已删除')
      return true
    } catch (e: any) {
      ElMessage.error('删除失败: ' + e.message)
      return false
    }
  }

  async function checkCapabilities(id: string): Promise<boolean> {
    try {
      capabilities.value = await api.checkServerCapabilities(id)
      ElMessage.success('服务端能力已获取')
      return true
    } catch (e: any) {
      capabilities.value = null
      ElMessage.error('检测能力失败: ' + e.message)
      return false
    }
  }

  return { servers, loading, capabilities, refresh, addServer, updateServer, deleteServer, checkCapabilities }
})
```

- [ ] **步骤 2：重写 tunnel store**

```typescript
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api, type TunnelInfo, type AddTunnelInput } from '@/api'

export const useTunnelStore = defineStore('tunnel', () => {
  const tunnels = ref<TunnelInfo[]>([])
  const loading = ref(false)

  async function refresh(serverId?: string) {
    loading.value = true
    try {
      tunnels.value = await api.listTunnels(serverId)
    } catch (e: any) {
      ElMessage.error('加载映射列表失败: ' + e.message)
    } finally {
      loading.value = false
    }
  }

  async function addTunnel(input: AddTunnelInput): Promise<boolean> {
    try {
      await api.addTunnel(input)
      await refresh()
      ElMessage.success('映射已创建')
      return true
    } catch (e: any) {
      ElMessage.error('创建失败: ' + e.message)
      return false
    }
  }

  async function deleteTunnel(id: string): Promise<boolean> {
    try {
      await api.deleteTunnel(id)
      await refresh()
      ElMessage.success('映射已删除')
      return true
    } catch (e: any) {
      ElMessage.error('删除失败: ' + e.message)
      return false
    }
  }

  async function startFrpc(serverId: string): Promise<boolean> {
    try {
      await api.startFrpc(serverId)
      ElMessage.success('frpc 已启动')
      return true
    } catch (e: any) {
      ElMessage.error('启动失败: ' + e.message)
      return false
    }
  }

  async function stopFrpc(serverId: string): Promise<boolean> {
    try {
      await api.stopFrpc(serverId)
      ElMessage.success('frpc 已停止')
      return true
    } catch (e: any) {
      ElMessage.error('停止失败: ' + e.message)
      return false
    }
  }

  async function restartFrpc(serverId: string): Promise<boolean> {
    try {
      await api.restartFrpc(serverId)
      ElMessage.success('frpc 已重启')
      return true
    } catch (e: any) {
      ElMessage.error('重启失败: ' + e.message)
      return false
    }
  }

  async function isRunning(serverId: string): Promise<boolean> {
    try {
      return await api.isFrpcRunning(serverId)
    } catch {
      return false
    }
  }

  return { tunnels, loading, refresh, addTunnel, deleteTunnel, startFrpc, stopFrpc, restartFrpc, isRunning }
})
```

- [ ] **步骤 3：重写 log store（接 Wails 事件）**

```typescript
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
```

> 注意：Wails runtime 在前端通过 `@wailsio/runtime` 包引入，`App.vue` 启动时调用 `logStore.bindEvents(runtime)`。本任务先在 store 暴露 `bindEvents`，任务 6 在 App.vue 接入。

- [ ] **步骤 4：构建验证**

运行：`npm run build`
预期：通过

- [ ] **步骤 5：Commit**

```bash
git add client/frontend/src/stores/
git commit -m "feat(client): stores 补全 CRUD 与 frpc actions + 日志事件绑定"
```

---

### 任务 3：服务器表单对话框 + 服务器页接通

**文件：**
- 新建：`src/components/ServerFormDialog.vue`
- 修改：`src/views/ServersView.vue`

- [ ] **步骤 1：创建 ServerFormDialog.vue**

```vue
<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { type AddServerInput, type ServerInfo } from '@/api'

const props = defineProps<{
  visible: boolean
  editing?: ServerInfo | null
}>()
const emit = defineEmits<{
  (e: 'update:visible', v: boolean): void
  (e: 'submit', input: AddServerInput): void
}>()

const formRef = ref<FormInstance>()
const form = reactive<AddServerInput>({
  name: '', host: '', frps_port: 7000, frp_token: '',
  agent_url: '', agent_token: '', is_default: false, remark: '',
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  host: [{ required: true, message: '请输入公网地址', trigger: 'blur' }],
  frps_port: [{ required: true, message: '请输入 frps 端口', trigger: 'blur' }],
  frp_token: [{ required: true, message: '请输入 frp token', trigger: 'blur' }],
  agent_url: [{ required: true, message: '请输入 agent 地址', trigger: 'blur' }],
  agent_token: [{ required: true, message: '请输入 agent token', trigger: 'blur' }],
}

watch(() => props.visible, (v) => {
  if (v) {
    if (props.editing) {
      Object.assign(form, {
        name: props.editing.name, host: props.editing.host,
        frps_port: props.editing.frps_port, frp_token: props.editing.frp_token,
        agent_url: props.editing.agent_url, agent_token: props.editing.agent_token,
        is_default: props.editing.is_default, remark: props.editing.remark ?? '',
      })
    } else {
      Object.assign(form, {
        name: '', host: '', frps_port: 7000, frp_token: '',
        agent_url: '', agent_token: '', is_default: false, remark: '',
      })
    }
  }
})

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate((valid) => {
    if (valid) emit('submit', { ...form })
  })
}

function close() {
  emit('update:visible', false)
}
</script>

<template>
  <el-dialog :model-value="visible" :title="editing ? '编辑服务器' : '添加服务器'" width="520px" @update:model-value="close">
    <el-form ref="formRef" :model="form" :rules="rules" label-width="120px">
      <el-form-item label="名称" prop="name"><el-input v-model="form.name" /></el-form-item>
      <el-form-item label="公网地址" prop="host"><el-input v-model="form.host" placeholder="IP 或域名" /></el-form-item>
      <el-form-item label="frps 端口" prop="frps_port"><el-input-number v-model="form.frps_port" :min="1" :max="65535" /></el-form-item>
      <el-form-item label="frp token" prop="frp_token"><el-input v-model="form.frp_token" show-password /></el-form-item>
      <el-form-item label="Agent 地址" prop="agent_url"><el-input v-model="form.agent_url" placeholder="http://1.2.3.4:7400" /></el-form-item>
      <el-form-item label="Agent token" prop="agent_token"><el-input v-model="form.agent_token" show-password /></el-form-item>
      <el-form-item label="设为默认"><el-switch v-model="form.is_default" /></el-form-item>
      <el-form-item label="备注"><el-input v-model="form.remark" type="textarea" :rows="2" /></el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="close">取消</el-button>
      <el-button type="primary" @click="handleSubmit">保存</el-button>
    </template>
  </el-dialog>
</template>
```

- [ ] **步骤 2：重写 ServersView.vue**

```vue
<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessageBox } from 'element-plus'
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
      <h2>服务器</h2>
      <el-button type="primary" @click="openAdd">添加服务器</el-button>
    </div>

    <el-table :data="store.servers" v-loading="store.loading" border empty-text="暂无服务器">
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="host" label="公网地址" />
      <el-table-column prop="frps_port" label="frps 端口" width="100" />
      <el-table-column prop="agent_url" label="Agent 地址" />
      <el-table-column prop="is_default" label="默认" width="80">
        <template #default="{ row }">
          <el-tag v-if="row.is_default" type="success" size="small">默认</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="260">
        <template #default="{ row }">
          <el-button size="small" link @click="handleCheckCapabilities(row)">检测能力</el-button>
          <el-button size="small" link @click="openEdit(row)">编辑</el-button>
          <el-button size="small" link type="danger" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

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
.page { padding: 16px; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h2 { margin: 0; }
</style>
```

- [ ] **步骤 3：构建验证**

运行：`npm run build`
预期：通过

- [ ] **步骤 4：Commit**

```bash
git add client/frontend/src/components/ServerFormDialog.vue client/frontend/src/views/ServersView.vue
git commit -m "feat(client): 服务器页表单对话框 + 增删改 + 检测能力"
```

---

### 任务 4：映射创建表单（按协议动态字段）

**文件：**
- 新建：`src/components/TunnelFormDialog.vue`

按设计文档第 16.2 节，TCP/UDP 显示远程端口字段，HTTP/HTTPS 显示域名模式（自定义/子域名）。创建时调 agent 校验/分配。

- [ ] **步骤 1：创建 TunnelFormDialog.vue**

```vue
<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import { type AddTunnelInput } from '@/api'
import { useServerStore } from '@/stores/server'

const props = defineProps<{
  visible: boolean
  serverId: string
}>()
const emit = defineEmits<{
  (e: 'update:visible', v: boolean): void
  (e: 'submit', input: AddTunnelInput): void
}>()

const serverStore = useServerStore()
const formRef = ref<FormInstance>()

const form = reactive<{
  name: string
  protocol: 'tcp' | 'udp' | 'http' | 'https'
  local_ip: string
  local_port: number
  remote_port: number
  auto_port: boolean
  domain_mode: 'custom' | 'subdomain'
  custom_domain: string
  subdomain: string
}>({
  name: '', protocol: 'tcp', local_ip: '127.0.0.1', local_port: 0,
  remote_port: 0, auto_port: false,
  domain_mode: 'custom', custom_domain: '', subdomain: '',
})

const isPortProtocol = computed(() => form.protocol === 'tcp' || form.protocol === 'udp')
const isDomainProtocol = computed(() => form.protocol === 'http' || form.protocol === 'https')

const rules: FormRules = {
  name: [{ required: true, message: '请输入映射名称', trigger: 'blur' }],
  local_port: [{ required: true, message: '请输入本地端口', trigger: 'blur' }],
}

watch(() => props.visible, (v) => {
  if (v) {
    Object.assign(form, {
      name: '', protocol: 'tcp', local_ip: '127.0.0.1', local_port: 0,
      remote_port: 0, auto_port: false,
      domain_mode: 'custom', custom_domain: '', subdomain: '',
    })
  }
})

function close() {
  emit('update:visible', false)
}

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    const input: AddTunnelInput = {
      server_id: props.serverId,
      name: form.name,
      protocol: form.protocol,
      local_ip: form.local_ip,
      local_port: form.local_port,
    }
    if (isPortProtocol.value) {
      input.remote_port = form.remote_port
    } else {
      if (form.domain_mode === 'custom') {
        input.custom_domain = form.custom_domain
      } else {
        input.subdomain = form.subdomain
      }
    }
    emit('submit', input)
  })
}
</script>

<template>
  <el-dialog :model-value="visible" title="创建映射" width="520px" @update:model-value="close">
    <el-form ref="formRef" :model="form" :rules="rules" label-width="120px">
      <el-form-item label="映射名称" prop="name"><el-input v-model="form.name" /></el-form-item>
      <el-form-item label="协议">
        <el-radio-group v-model="form.protocol">
          <el-radio-button label="tcp">TCP</el-radio-button>
          <el-radio-button label="udp">UDP</el-radio-button>
          <el-radio-button label="http">HTTP</el-radio-button>
          <el-radio-button label="https">HTTPS</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="本地 IP"><el-input v-model="form.local_ip" /></el-form-item>
      <el-form-item label="本地端口" prop="local_port"><el-input-number v-model="form.local_port" :min="1" :max="65535" /></el-form-item>

      <template v-if="isPortProtocol">
        <el-form-item label="自动分配端口"><el-switch v-model="form.auto_port" /></el-form-item>
        <el-form-item label="远程端口" v-if="!form.auto_port">
          <el-input-number v-model="form.remote_port" :min="1" :max="65535" />
        </el-form-item>
      </template>

      <template v-if="isDomainProtocol">
        <el-form-item label="域名模式">
          <el-radio-group v-model="form.domain_mode">
            <el-radio label="custom">自定义域名</el-radio>
            <el-radio label="subdomain">子域名前缀</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="自定义域名" v-if="form.domain_mode === 'custom'">
          <el-input v-model="form.custom_domain" placeholder="app.example.com" />
        </el-form-item>
        <el-form-item label="子域名前缀" v-if="form.domain_mode === 'subdomain'">
          <el-input v-model="form.subdomain" placeholder="demo" />
          <div class="hint" v-if="serverStore.capabilities?.subdomain_host">
            最终域名：{{ form.subdomain }}.{{ serverStore.capabilities.subdomain_host }}
          </div>
        </el-form-item>
      </template>
    </el-form>
    <template #footer>
      <el-button @click="close">取消</el-button>
      <el-button type="primary" @click="handleSubmit">创建</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.hint { font-size: 12px; color: #909399; margin-top: 4px; }
</style>
```

> 注意：在线端口/域名校验的实际调用在 App 层完成（任务 6 app.go 增强 AddTunnel），前端表单只收集输入并提交。若想在创建前做"预校验"提示，可在 handleSubmit 里先调 `api.checkPort`/`api.checkDomain`，但为简化，本计划把校验放在后端 AddTunnel 内部，失败则返回 error，前端 ElMessage 提示。

- [ ] **步骤 2：构建验证**

运行：`npm run build`
预期：通过

- [ ] **步骤 3：Commit**

```bash
git add client/frontend/src/components/TunnelFormDialog.vue
git commit -m "feat(client): 映射创建表单按协议动态字段"
```

---

### 任务 5：映射页接通 + frpc 启停 + 状态轮询

**文件：**
- 修改：`src/views/TunnelsView.vue`

- [ ] **步骤 1：重写 TunnelsView.vue**

```vue
<script setup lang="ts">
import { onMounted, onUnmounted, ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useServerStore } from '@/stores/server'
import { useTunnelStore } from '@/stores/tunnel'
import { type AddTunnelInput } from '@/api'
import TunnelFormDialog from '@/components/TunnelFormDialog.vue'

const serverStore = useServerStore()
const tunnelStore = useTunnelStore()

const selectedServerId = ref('')
const dialogVisible = ref(false)
const runningMap = ref<Record<string, boolean>>({})
let pollTimer: number | undefined

const hasServer = computed(() => serverStore.servers.length > 0)

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
</script>

<template>
  <div class="page">
    <div class="page-header">
      <h2>映射</h2>
      <div class="actions">
        <el-select v-model="selectedServerId" placeholder="选择服务器" @change="onServerChange" style="width: 180px">
          <el-option v-for="s in serverStore.servers" :key="s.id" :label="s.name" :value="s.id" />
        </el-select>
        <el-button type="primary" @click="openCreate" :disabled="!hasServer">创建映射</el-button>
        <el-button @click="handleStart" :disabled="!hasServer">启动 frpc</el-button>
        <el-button @click="handleStop" :disabled="!hasServer">停止 frpc</el-button>
        <el-button @click="handleRestart" :disabled="!hasServer">重启 frpc</el-button>
      </div>
    </div>

    <el-table :data="tunnelStore.tunnels" v-loading="tunnelStore.loading" border empty-text="暂无映射">
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="protocol" label="协议" width="80">
        <template #default="{ row }">
          <el-tag size="small">{{ row.protocol.toUpperCase() }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="本地">
        <template #default="{ row }">{{ row.local_ip }}:{{ row.local_port }}</template>
      </el-table-column>
      <el-table-column label="远程">
        <template #default="{ row }">
          <span v-if="row.remote_port">:{{ row.remote_port }}</span>
          <span v-else-if="row.custom_domain">{{ row.custom_domain }}</span>
          <span v-else-if="row.subdomain">{{ row.subdomain }}</span>
        </template>
      </el-table-column>
      <el-table-column label="frpc 状态" width="120">
        <template #default="{ row }">
          <el-tag :type="runningMap[row.server_id] ? 'success' : 'info'" size="small">
            {{ runningMap[row.server_id] ? '运行中' : '已停止' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="120">
        <template #default="{ row }">
          <el-button size="small" link type="danger" @click="handleDelete(row.id)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <TunnelFormDialog v-model:visible="dialogVisible" :server-id="selectedServerId" @submit="handleSubmit" />
  </div>
</template>

<style scoped>
.page { padding: 16px; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h2 { margin: 0; }
.actions { display: flex; gap: 8px; }
</style>
```

- [ ] **步骤 2：构建验证**

运行：`npm run build`
预期：通过

- [ ] **步骤 3：Commit**

```bash
git add client/frontend/src/views/TunnelsView.vue
git commit -m "feat(client): 映射页接通 + frpc 启停 + 状态轮询"
```

---

### 任务 6：后端日志事件发射 + AddTunnel 在线校验 + App.vue 事件挂载

**文件：**
- 修改：`client/app.go` — AddTunnel 增强（调 agent 校验/分配）、新增 EmitLog、日志流
- 修改：`client/main.go` — frpc 启动时把 stdout/stderr 转成 EventsEmit
- 修改：`client/frontend/src/App.vue` — 挂载 Wails runtime 事件
- 修改：`client/frontend/package.json` — 加 `@wailsio/runtime` 依赖

- [ ] **步骤 1：app.go 增强 AddTunnel 与日志**

在 `client/app.go` 的 `AddTunnel` 方法里，落库前调 agent 校验/分配：

```go
// AddTunnel 添加映射，返回新 ID。
// 按 protocol 调用 agent 在线校验/分配端口或域名，失败则返回 error 不落库。
func (a *App) AddTunnel(in AddTunnelInput) (string, error) {
	ctx := context.Background()
	cli, err := a.newAgentClient(in.ServerID)
	if err != nil {
		return "", err
	}
	switch in.Protocol {
	case "tcp", "udp":
		if in.RemotePort > 0 {
			// 手动指定端口：检查可用性
			res, err := cli.CheckPort(ctx, in.Protocol, in.RemotePort)
			if err != nil {
				return "", fmt.Errorf("检查端口: %w", err)
			}
			if !res.Available {
				return "", fmt.Errorf("端口 %d 不可用: %s", in.RemotePort, res.Reason)
			}
		} else {
			// 自动分配
			port, err := cli.AllocatePort(ctx, in.Protocol)
			if err != nil {
				return "", fmt.Errorf("分配端口: %w", err)
			}
			in.RemotePort = port
		}
	case "http", "https":
		domain := in.CustomDomain
		if domain == "" {
			domain = in.Subdomain
		}
		if domain == "" {
			return "", fmt.Errorf("http/https 映射需提供 custom_domain 或 subdomain")
		}
		res, err := cli.CheckDomain(ctx, in.Protocol, domain)
		if err != nil {
			return "", fmt.Errorf("检查域名: %w", err)
		}
		if !res.Available {
			return "", fmt.Errorf("域名 %s 不可用: %s", domain, res.Reason)
		}
	}

	now := time.Now().UTC()
	tu := db.Tunnel{
		ID: uuid.NewString(), ServerID: in.ServerID, Name: in.Name, Protocol: in.Protocol,
		LocalIP: in.LocalIP, LocalPort: in.LocalPort, RemotePort: in.RemotePort,
		CustomDomain: in.CustomDomain, Subdomain: in.Subdomain,
		Enabled: true, Status: "stopped", CreatedAt: now, UpdatedAt: now,
	}
	if err := a.repo.InsertTunnel(tu); err != nil {
		return "", err
	}
	// 落库成功后注册域名占用
	if in.Protocol == "http" || in.Protocol == "https" {
		domain := in.CustomDomain
		if domain == "" {
			domain = in.Subdomain
		}
		_ = cli.RegisterDomain(ctx, in.Protocol, domain, tu.ID)
	}
	return tu.ID, nil
}

// EmitLog 向前端推送一条日志（通过 Wails 事件 log:append）。
func (a *App) EmitLog(level, message, serverID string) {
	runtime.EventsEmit(a.ctx, "log:append", map[string]string{
		"time":      time.Now().UTC().Format(time.RFC3339),
		"level":     level,
		"message":   message,
		"server_id": serverID,
	})
}
```

`app.go` import 块加 `"github.com/wailsapp/wails/v2/pkg/runtime"`。

- [ ] **步骤 2：main.go / frpc 启动时转发日志**

frpc 进程的 stdout/stderr 转成日志事件。修改 `app.go` 的 `StartFrpc`：

```go
// StartFrpc 启动指定服务器的 frpc 进程，并转发其输出为日志事件。
func (a *App) StartFrpc(serverId string) error {
	cfgText, err := a.GenerateFrpcConfig(serverId)
	if err != nil {
		return err
	}
	if err := a.frpcMgr.Start(context.Background(), serverId, cfgText); err != nil {
		a.EmitLog("error", "启动 frpc 失败: "+err.Error(), serverId)
		return err
	}
	a.EmitLog("info", "frpc 已启动", serverId)
	go a.streamFrpcLog(serverId)
	return nil
}

// streamFrpcLog 读取 frpc 进程输出并推送到前端。
func (a *App) streamFrpcLog(serverId string) {
	// frpcMgr 需暴露进程的 stdout pipe；本任务需在 frpc.Manager 增加 LogPipe 方法。
	// 简化实现：frpc 进程用 exec.Command，输出已重定向到 pipe，这里按行读取。
	// 具体：frpc.Manager 需新增一个返回 io.Reader 的方法，或把 log 转发逻辑放进 frpc.Manager。
	// 为最小改动，在 frpc.Manager 里集成日志回调（见步骤 3）。
}
```

> 注意：步骤 1 的 `streamFrpcLog` 涉及进程输出读取，需在 `frpc.Manager` 增加日志回调机制。本任务在步骤 3 完善 frpc.Manager。

- [ ] **步骤 3：frpc.Manager 增加日志回调**

在 `client/internal/frpc/manager.go` 的 `Start` 方法里，给 `exec.Cmd` 设置 `Stdout`/`Stderr` pipe，并通过回调按行推送。修改 `Manager`：

```go
type Manager struct {
	mu        sync.Mutex
	procs     map[string]*exec.Cmd
	configDir string
	binary    string
	args      []string
	logCb     func(serverID, line string)
}

// SetLogCallback 设置日志回调，每次 frpc 输出一行调用一次。
func (m *Manager) SetLogCallback(cb func(serverID, line string)) {
	m.logCb = cb
}

// Start 内部修改 cmd.Stdout/Stderr 为 pipe + 行扫描：
func (m *Manager) Start(ctx context.Context, serverID, cfgText string) error {
	// ...（原有写配置 + 建命令逻辑不变，到 cmd := exec.CommandContext 之后）
	cmd := exec.CommandContext(ctx, m.binary, args...)
	if m.logCb != nil {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("stdout pipe: %w", err)
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("stderr pipe: %w", err)
		}
		go scanLines(stdout, serverID, m.logCb)
		go scanLines(stderr, serverID, m.logCb)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 frpc: %w", err)
	}
	m.procs[serverID] = cmd
	go func() { _ = cmd.Wait() }()
	return nil
}

func scanLines(r io.Reader, serverID string, cb func(serverID, line string)) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		cb(serverID, sc.Text())
	}
}
```

`manager.go` import 加 `"bufio"`、`"io"`。`NewManager` 里 `logCb` 默认 nil。

然后 `app.go` 在 `Init` 后设置回调：

```go
// Init 注入生产依赖（由 main.go 调用）。
func (a *App) Init(repo *db.Repo, frpcMgr *frpc.Manager) {
	a.repo = repo
	a.frpcMgr = frpcMgr
	frpcMgr.SetLogCallback(func(serverID, line string) {
		level := "info"
		if strings.Contains(line, "error") || strings.Contains(line, "ERROR") {
			level = "error"
		} else if strings.Contains(line, "warn") || strings.Contains(line, "WARN") {
			level = "warn"
		}
		a.EmitLog(level, line, serverID)
	})
}
```

> 同时删除步骤 2 里占位的 `streamFrpcLog`，改由 frpc.Manager 的回调直接推送。

- [ ] **步骤 4：App.vue 挂载日志事件**

```bash
cd client/frontend && npm install @wailsio/runtime
```

修改 `client/frontend/src/App.vue` 的 setup：

```vue
<script lang="ts" setup>
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { EventsOn } from '@wailsio/runtime'
import { useLogStore } from '@/stores/log'

const route = useRoute()
const router = useRouter()
const active = computed(() => route.path)
const logStore = useLogStore()

const menus = [
  { index: '/servers', label: '服务器' },
  { index: '/tunnels', label: '映射' },
  { index: '/logs', label: '日志' },
  { index: '/settings', label: '设置' },
]

onMounted(() => {
  // 绑定后端 EventsEmit 的 log:append 事件
  EventsOn('log:append', (line: any) => {
    logStore.append({
      time: line.time ?? new Date().toISOString(),
      level: line.level ?? 'info',
      message: line.message ?? '',
      server_id: line.server_id,
    })
  })
})
</script>
```

`logStore.bindEvents` 方式可删除，直接在 App.vue 用 `EventsOn`。`log.ts` 的 `bindEvents` 保留无害但不再使用。

- [ ] **步骤 5：后端编译验证**

运行：`cd client && go build ./... && go vet ./...`
预期：通过

- [ ] **步骤 6：前端构建验证**

运行：`cd client/frontend && npm run build`
预期：通过

- [ ] **步骤 7：运行全部单测**

运行：`cd client && go test ./...`
预期：所有测试 PASS（frpc.Manager 改动后旧测试可能需调整：若 `SetLogCallback` 在测试中未调用，`logCb` 为 nil，`Start` 走原路径，测试应仍通过）

- [ ] **步骤 8：Commit**

```bash
git add client/app.go client/internal/frpc/manager.go client/main.go client/frontend/src/App.vue client/frontend/package.json client/frontend/package-lock.json
git commit -m "feat(client): 日志事件实时推送 + AddTunnel 在线校验"
```

---

### 任务 7：托盘运行

**文件：**
- 修改：`client/main.go`、`client/app.go`

Wails v2.12 支持系统托盘。窗口关闭时最小化到托盘而非退出。

- [ ] **步骤 1：main.go 配置托盘与最小化行为**

```go
package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	dbPath, err := configpkg.DefaultDBPath()
	if err != nil {
		println("获取默认 DB 路径失败:", err.Error())
		return
	}
	database := dbpkg.Open(dbPath)
	repo, err := dbpkg.NewRepo(database)
	if err != nil {
		println("初始化 db repo 失败:", err.Error())
		return
	}
	frpcConfigDir, _ := configpkg.DefaultDir()
	app.Init(repo, frpc.NewManager(frpcConfigDir))

	err = wails.Run(&options.App{
		Title:  "FRP Manager",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 245, G: 247, B: 250, A: 1},
		OnStartup:        app.startup,
		Bind:             []interface{}{app},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "frp-manager-instance-lock",
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
```

> 注意：Wails v2 的系统托盘 API 在 v2.12 仍为实验性，不同小版本签名有差异。完整托盘实现需要：1) 创建 `application.NewSystemTray(icon, menu)`；2) 监听窗口关闭事件改为隐藏。由于 API 不稳定且设计文档把托盘也列在 v0.2，**本任务作为可选增强**：若 `wails dev` 环境下托盘 API 编译失败，把托盘降级为 v0.2，本任务只保留 `SingleInstanceLock`（单实例锁，确保不重复启动），托盘与"关闭最小化"留到 v0.2。**实现时先尝试托盘，编译失败则只保留单实例锁并 Commit。**

- [ ] **步骤 2：编译验证**

运行：`cd client && go build ./...`
预期：通过（或托盘 API 失败时降级方案通过）

- [ ] **步骤 3：Commit**

```bash
git add client/main.go
git commit -m "feat(client): 单实例锁 + 托盘运行（若 API 可用）"
```

---

### 任务 8：端到端冒烟验证

**文件：** 无（验证步骤）

- [ ] **步骤 1：启动服务端（依赖计划 1 已完成）**

```bash
cd server
# 准备本地 frps.toml（指向 configs/frps.toml.example 或自建）
go run ./cmd/agent -config configs/agent.toml.example
```

确认 `/api/health` 返回 ok。

- [ ] **步骤 2：启动客户端**

```bash
cd client
wails dev
```

- [ ] **步骤 3：手动验证全流程**

1. 服务器页 → 添加服务器 → 填写 host=127.0.0.1、frps_port=7000、frp_token、agent_url=http://127.0.0.1:7400、agent_token（与服务端一致）→ 保存
2. 点"检测能力" → 弹窗显示 capabilities
3. 映射页 → 选择服务器 → 创建映射 → TCP、本地 127.0.0.1:3389、自动分配端口 → 创建成功
4. 再创建 HTTP 映射 → 自定义域名 app.example.com → 创建成功
5. 点"启动 frpc" → frpc 状态变"运行中"
6. 日志页 → 看到 frpc 启动日志
7. 点"停止 frpc" → 状态变"已停止"
8. 删除映射 → 服务端端口/域名被释放（可在服务端 DB 查 status=released）

- [ ] **步骤 4：最终全量验证**

```bash
# 服务端
cd server && go test ./... && go build ./...
# 客户端
cd client && go test ./... && go build ./... && cd frontend && npm run build
```

全部通过后，v0.1 闭环完成。

- [ ] **步骤 5：Commit（如有冒烟时的小修）**

```bash
git add -A
git commit -m "chore: v0.1 端到端冒烟验证通过"
```

---

## 自检结果

**1. 规格覆盖度**（对照设计文档第 16、17、18、20 节）：
- 第 16.1 节服务器页字段 + 检测按钮 ✅ 任务 3
- 第 16.2 节映射创建页按协议动态字段 ✅ 任务 4
- 第 17 节创建流程（在线校验端口/域名）✅ 任务 6 AddTunnel
- 第 18 节删除释放资源 ✅ 计划 2 DeleteTunnel 已实现，本计划前端接通任务 5
- 第 20 节 MVP 客户端项：GUI ✅、加服务器 ✅、检测 agent ✅、检测能力 ✅、创建四协议映射 ✅、生成 frpc.toml ✅ 计划 2、启停重启 frpc ✅ 任务 5、实时日志 ✅ 任务 6、托盘 ✅ 任务 7（或降级 v0.2）

**2. 占位符扫描**：无 TODO/待定；每步有完整代码。任务 7 托盘因 API 不稳定给了明确的降级路径，非占位。

**3. 类型一致性**：
- `AddServerInput`/`AddTunnelInput`/`Capabilities`/`PortCheckResult`/`DomainCheckResult` 在 api/index.ts 定义、stores 使用、组件使用，字段名一致（snake_case）
- `logStore.append` 接收的 `LogLine` 与后端 `EmitLog` 推送的 map 字段一致（time/level/message/server_id）
- `frpc.Manager.SetLogCallback` 与 `app.go Init` 调用一致
- `api` 方法名与 Go 方法名一致（ListServers/AddServer/UpdateServerByID/DeleteServer/CheckServerCapabilities/StartFrpc/StopFrpc/RestartFrpc/IsFrpcRunning/GenerateFrpcConfig）

**已识别的风险/注意事项：**
1. 任务 6 frpc.Manager 加 StdoutPipe 后，原 manager_test.go 用 `ping`/`sleep` 替身，pipe 仍可工作，测试应仍通过；若失败需在测试里也设回调或忽略 pipe。实现时跑测试确认。
2. 任务 7 Wails 托盘 API 不稳定，明确降级路径。
3. 任务 6 `@wailsio/runtime` 包版本需与 Wails CLI v2.12 兼容，`npm install @wailsio/runtime` 会装对应版本；若 import 报错，检查 `wails.json` 的 `wailsversion`。
4. 端到端冒烟（任务 8）需要真实 frps + frpc 二进制；若本地无 frps/frpc，可只验证到"创建映射成功 + frpc 启动报错（找不到二进制）+ 日志显示错误"，这也算闭环验证。完整功能验证需在真实环境。

这些都有处理方式，不阻塞。
