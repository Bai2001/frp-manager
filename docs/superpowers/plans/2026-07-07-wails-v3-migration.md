# Wails v2 → v3 迁移实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将客户端从 Wails v2.12.0 迁移到 v3.0.0-alpha2.115，保持 GUI + CRUD + frpc 启停 + 实时日志全部可用。

**架构：** App 结构体改造为 v3 Service（ServiceStartup/ServiceShutdown 生命周期），main.go 用 `application.New` + 独立窗口创建 + `app.Run()`；前端从 `window.go` + `wailsjs/runtime` 切换到 `@wailsio/runtime` + 自动生成的 `bindings/` 绑定；配置文件从 `wails.json` 切换到 `Taskfile.yml` + `build/config.yml`。

**技术栈：** Go 1.26、Wails v3.0.0-alpha2.115、Vue3 + TypeScript + Element Plus + Pinia + Vue Router、Vite 6 + `@wailsio/runtime`、modernc.org/sqlite

**参照项目：** `C:\Users\15953\code\other\wails3-ref`（已用 `wails3 init -n wails3-ref -t vue` 创建的 v3 模板项目，用于对照配置文件和目录结构）

---

## 文件结构

**Go 端（client/）：**
- 修改：`client/go.mod` — 依赖 v2 → v3
- 修改：`client/main.go` — application.New + 窗口创建 + app.Run
- 修改：`client/app.go` — ServiceStartup/ServiceShutdown + SetApplication + EmitLog 用 app.Event
- 修改：`client/app_test.go` — 适配 InitForTest（如有需要）
- 修改：`client/types.go` — 无需改动（确认）

**配置文件（client/）：**
- 删除：`client/wails.json`
- 创建：`client/Taskfile.yml` — v3 任务定义
- 创建：`client/build/config.yml` — v3 dev_mode 配置
- 修改：`client/build/windows/info.json` — 填入项目实际信息
- 修改：`client/build/windows/installer/project.nsi` — 替换为 v3 nsis 模板（从 wails3-ref 复制）
- 创建：`client/build/windows/nsis/project.nsi` + `wails_tools.nsh`（v3 nsis 结构变化）
- 删除：`client/build/darwin/`（v3 结构不同，本次 Windows 优先，darwin 后续按需重建）

**前端（client/frontend/）：**
- 删除：`client/frontend/wailsjs/` — 旧 v2 绑定目录
- 创建：`client/frontend/bindings/` — v3 自动生成（由 `wails3 generate bindings -ts` 产生）
- 修改：`client/frontend/package.json` — 加 `@wailsio/runtime` 依赖
- 修改：`client/frontend/vite.config.ts` — 加 wails 插件
- 修改：`client/frontend/tsconfig.json` — include 改为 bindings
- 修改：`client/frontend/src/api/index.ts` — import 绑定替换 window.go
- 修改：`client/frontend/src/App.vue` — Events.On 替换 EventsOn
- 修改：`client/frontend/src/stores/log.ts` — Events.On 替换 EventsOn

---

## 任务 1：Go 依赖切换到 v3

**文件：**
- 修改：`client/go.mod`

- [ ] **步骤 1：移除 v2 依赖，添加 v3 依赖**

编辑 `client/go.mod`，将 `require` 块中的：
```
github.com/wailsapp/wails/v2 v2.12.0
```
替换为：
```
github.com/wailsapp/wails/v3 v3.0.0-alpha2.115
```

- [ ] **步骤 2：更新 go.sum 并拉取依赖**

运行：
```bash
cd client
go mod tidy
```
预期：下载 v3 依赖，移除 v2 相关条目。若网络问题，先设 `$env:GOPROXY="https://goproxy.cn,direct"`。

- [ ] **步骤 3：验证依赖切换成功**

运行：
```bash
cd client
go list -m github.com/wailsapp/wails/v3
```
预期：输出 `github.com/wailsapp/wails/v3 v3.0.0-alpha2.115`

- [ ] **步骤 4：Commit**

```bash
git add client/go.mod client/go.sum
git commit -m "chore(client): 切换 wails 依赖到 v3.0.0-alpha2.115"
```

---

## 任务 2：app.go 改造为 v3 Service

**文件：**
- 修改：`client/app.go`

- [ ] **步骤 1：更新 import 块**

将 `client/app.go` 第 11-18 行的 import 块：
```go
import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kdc/frp-manager/client/internal/agent"
	"github.com/kdc/frp-manager/client/internal/db"
	"github.com/kdc/frp-manager/client/internal/frpc"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)
```
替换为：
```go
import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kdc/frp-manager/client/internal/agent"
	"github.com/kdc/frp-manager/client/internal/config"
	"github.com/kdc/frp-manager/client/internal/db"
	"github.com/kdc/frp-manager/client/internal/frpc"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/wailsapp/wails/v3/pkg/application"
)
```

- [ ] **步骤 2：App 结构体加 app 字段**

将 App 结构体定义（约第 20-27 行）：
```go
type App struct {
	ctx      context.Context
	database *sql.DB
	repo     *db.Repo
	frpcMgr  *frpc.Manager
}
```
替换为：
```go
type App struct {
	app      *application.App
	ctx      context.Context
	database *sql.DB
	repo     *db.Repo
	frpcMgr  *frpc.Manager
}
```

- [ ] **步骤 3：新增 SetApplication 方法，删除旧 startup 方法**

将旧的 `startup` 方法（约第 36-39 行）：
```go
// startup 在应用启动时由 Wails 调用。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}
```
替换为：
```go
// SetApplication 注入 Wails v3 应用实例（由 main.go 在 application.New 后回填）。
// 供 EmitLog 等通过 app.Event.Emit 推送事件到前端。
func (a *App) SetApplication(app *application.App) {
	a.app = app
}

// ServiceStartup 在应用启动时由 Wails v3 调用。
// 初始化 db/repo/frpcMgr，返回 error 可中断启动。
func (a *App) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	a.ctx = ctx
	dbPath, err := config.DefaultDBPath()
	if err != nil {
		return fmt.Errorf("获取默认 DB 路径: %w", err)
	}
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("打开客户端数据库: %w", err)
	}
	repo, err := db.NewRepo(database)
	if err != nil {
		_ = database.Close()
		return fmt.Errorf("初始化 db repo: %w", err)
	}
	a.database = database
	a.repo = repo
	a.frpcMgr = frpc.NewManager()
	a.frpcMgr.SetLogCallback(func(serverID, line string) {
		level := "info"
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") {
			level = "error"
		} else if strings.Contains(lower, "warn") {
			level = "warn"
		}
		a.EmitLog(level, line, serverID)
	})
	return nil
}

// ServiceShutdown 在应用退出时由 Wails v3 调用，释放数据库连接。
func (a *App) ServiceShutdown() error {
	if a.database != nil {
		return a.database.Close()
	}
	return nil
}
```

- [ ] **步骤 4：简化 Init 方法（移除生产依赖注入，改由 ServiceStartup 负责）**

将旧的 `Init` 方法（约第 41-55 行）：
```go
// Init 注入生产依赖（由 main.go 调用）。
// 同时给 frpc.Manager 设置日志回调，把 frpc 进程输出转发为前端日志事件。
func (a *App) Init(repo *db.Repo, frpcMgr *frpc.Manager) {
	a.repo = repo
	a.frpcMgr = frpcMgr
	frpcMgr.SetLogCallback(func(serverID, line string) {
		level := "info"
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") {
			level = "error"
		} else if strings.Contains(lower, "warn") {
			level = "warn"
		}
		a.EmitLog(level, line, serverID)
	})
}
```
替换为（仅保留测试用的依赖注入，生产依赖由 ServiceStartup 负责）：
```go
// InitForTest 注入测试依赖，不经过 Wails 运行时。
func (a *App) InitForTest(dbPath string) error {
	d, err := db.Open(dbPath)
	if err != nil {
		return err
	}
	r, err := db.NewRepo(d)
	if err != nil {
		_ = d.Close()
		return err
	}
	a.database = d
	a.repo = r
	a.frpcMgr = frpc.NewManager()
	return nil
}
```
**注意：** 删除旧的 `Init` 方法（已被 ServiceStartup 取代）和旧的 `InitForTest` 方法（被这个新版本替换）。

- [ ] **步骤 5：保留 SetDatabase 和 Close 方法**

`SetDatabase` 方法可删除（v3 用 ServiceShutdown 关库），但 `Close` 方法保留供测试 Cleanup 调用。将：
```go
// SetDatabase 注入生产环境的底层数据库连接（供退出时关闭）。
func (a *App) SetDatabase(d *sql.DB) {
	a.database = d
}

// InitForTest 注入测试依赖。
func (a *App) InitForTest(dbPath string) error {
	d, err := db.Open(dbPath)
	if err != nil {
		return err
	}
	r, err := db.NewRepo(d)
	if err != nil {
		return err
	}
	a.database = d
	a.repo = r
	a.frpcMgr = frpc.NewManager()
	return nil
}
```
替换为（仅保留 Close，删除 SetDatabase 和旧 InitForTest）：
```go
// Close 释放 App 持有的资源（供测试清理调用）。
// 生产环境由 ServiceShutdown 负责。
func (a *App) Close() {
	if a.database != nil {
		_ = a.database.Close()
	}
}
```

- [ ] **步骤 6：改造 EmitLog 用 app.Event.Emit**

将 `EmitLog` 方法（约第 232-241 行）：
```go
// EmitLog 向前端推送一条日志（通过 Wails 事件 log:append）。
func (a *App) EmitLog(level, message, serverID string) {
	if a.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(a.ctx, "log:append", map[string]string{
		"time":      time.Now().UTC().Format(time.RFC3339),
		"level":     level,
		"message":   message,
		"server_id": serverID,
	})
}
```
替换为：
```go
// EmitLog 向前端推送一条日志（通过 Wails v3 事件 log:append）。
func (a *App) EmitLog(level, message, serverID string) {
	if a.app == nil {
		return
	}
	a.app.Event.Emit("log:append", map[string]string{
		"time":      time.Now().UTC().Format(time.RFC3339),
		"level":     level,
		"message":   message,
		"server_id": serverID,
	})
}
```

- [ ] **步骤 7：验证编译**

运行：
```bash
cd client
go build ./...
```
预期：编译失败（main.go 还没改，引用了旧 API），但 app.go 本身无语法错误。记录错误用于下一步。

- [ ] **步骤 8：Commit**

```bash
git add client/app.go
git commit -m "refactor(client): App 改造为 Wails v3 Service（ServiceStartup/Shutdown + app.Event）"
```

---

## 任务 3：main.go 改造为 v3 应用结构

**文件：**
- 修改：`client/main.go`

- [ ] **步骤 1：重写 main.go**

将 `client/main.go` 全部内容替换为：
```go
package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	appStruct := NewApp()

	app := application.New(application.Options{
		Name:        "FRP Manager",
		Description: "基于 frp 的本地 GUI 内网穿透管理系统",
		Services: []application.Service{
			application.NewService(appStruct),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "frp-manager-instance-lock",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				// 第二实例启动时，把已运行实例的窗口前置显示
				if w := app.Window.Current(); w != nil {
					w.Show()
					w.Focus()
				}
			},
		},
	})

	// 回填 app 引用，供 EmitLog 等使用 app.Event.Emit
	appStruct.SetApplication(app)

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "FRP Manager",
		Width:            1024,
		Height:           768,
		BackgroundColour: application.NewRGB(245, 247, 250),
		URL:              "/",
	})
	window.Show()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **步骤 2：验证编译**

运行：
```bash
cd client
go build ./...
```
预期：编译成功。若失败，根据错误调整（常见：import 路径、API 签名差异）。

- [ ] **步骤 3：Commit**

```bash
git add client/main.go
git commit -m "refactor(client): main.go 改造为 Wails v3 application.New + 独立窗口创建"
```

---

## 任务 4：运行 Go 单测确认 app 改造未破坏测试

**文件：**
- 验证：`client/app_test.go`

- [ ] **步骤 1：运行客户端全部测试**

运行：
```bash
cd client
go test ./...
```
预期：全部 PASS。`app_test.go` 走 `InitForTest` 直接注入依赖，不经过 Wails 运行时，应不受影响。`EmitLog` 的 nil 守卫从 `a.ctx == nil` 改为 `a.app == nil`，测试中 `a.app` 为 nil 即跳过事件发射，行为一致。

- [ ] **步骤 2：若有测试失败则修复**

若 `InitForTest` 签名变化导致测试编译失败，调整 `app_test.go` 中的 `newTestApp` 函数适配新签名。预期无需改动。

- [ ] **步骤 3：Commit（如有测试文件改动）**

```bash
git add client/app_test.go
git commit -m "test(client): 适配 Wails v3 Service 改造"
```
（若测试无改动则跳过此 commit）

---

## 任务 5：前端依赖与 Vite 配置切换

**文件：**
- 修改：`client/frontend/package.json`
- 修改：`client/frontend/vite.config.ts`
- 修改：`client/frontend/tsconfig.json`
- 删除：`client/frontend/wailsjs/`

- [ ] **步骤 1：更新 package.json**

将 `client/frontend/package.json` 的 `dependencies` 块：
```json
  "dependencies": {
    "axios": "^1.7.9",
    "element-plus": "^2.9.4",
    "pinia": "^2.3.1",
    "vue": "^3.5.13",
    "vue-router": "^4.5.0"
  },
```
替换为（加 `@wailsio/runtime`）：
```json
  "dependencies": {
    "@wailsio/runtime": "alpha",
    "axios": "^1.7.9",
    "element-plus": "^2.9.4",
    "pinia": "^2.3.1",
    "vue": "^3.5.13",
    "vue-router": "^4.5.0"
  },
```

同时将 `scripts.build`：
```json
    "build": "vue-tsc --noEmit && vite build",
```
替换为（对齐 v3 模板，--noEmit 改为类型检查模式）：
```json
    "build": "vue-tsc && vite build --mode production",
```

- [ ] **步骤 2：更新 vite.config.ts**

将 `client/frontend/vite.config.ts` 全部内容替换为：
```ts
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import wails from '@wailsio/runtime/plugins/vite'
import path from 'path'

export default defineConfig({
  server: {
    host: '127.0.0.1',
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
    },
  },
  plugins: [vue(), wails('./bindings')],
})
```

- [ ] **步骤 3：更新 tsconfig.json 的 include**

将 `client/frontend/tsconfig.json` 的 `include` 块：
```json
  "include": [
    "src/**/*.ts",
    "src/**/*.d.ts",
    "src/**/*.tsx",
    "src/**/*.vue",
    "wailsjs/**/*.ts"
  ],
```
替换为（wailsjs → bindings）：
```json
  "include": [
    "src/**/*.ts",
    "src/**/*.d.ts",
    "src/**/*.tsx",
    "src/**/*.vue",
    "bindings/**/*.ts"
  ],
```

- [ ] **步骤 4：删除旧 wailsjs 目录**

运行：
```bash
cd client/frontend
Remove-Item -Recurse -Force wailsjs
```
预期：`client/frontend/wailsjs/` 目录删除。

- [ ] **步骤 5：安装新依赖**

运行：
```bash
cd client/frontend
npm install
```
预期：安装 `@wailsio/runtime`。若网络问题，先 `npm config set registry https://registry.npmmirror.com`。

- [ ] **步骤 6：Commit**

```bash
git add client/frontend/package.json client/frontend/vite.config.ts client/frontend/tsconfig.json
git rm -r client/frontend/wailsjs
git commit -m "chore(client/frontend): 切换到 @wailsio/runtime + wails vite 插件，删除旧 wailsjs"
```

---

## 任务 6：生成 v3 绑定

**文件：**
- 创建：`client/frontend/bindings/`（自动生成）

- [ ] **步骤 1：生成 TypeScript 绑定**

运行：
```bash
cd client
wails3 generate bindings -ts
```
预期：输出 `Processed: N Packages, 1 Service, M Methods...`，生成到 `client/frontend/bindings/`。绑定按 module path 分组，路径应为 `frontend/bindings/github.com/kdc/frp-manager/client/`。

- [ ] **步骤 2：确认绑定目录结构**

运行：
```bash
ls client/frontend/bindings
```
预期：看到 `github.com/kdc/frp-manager/client/` 目录，内含 `app.ts`（App 服务的方法绑定）和 `index.ts`。

- [ ] **步骤 3：检查生成的 app.ts**

读取 `client/frontend/bindings/github.com/kdc/frp-manager/client/app.ts`，确认包含 `ListServers`、`AddServer`、`StartFrpc` 等方法绑定。每个方法形如：
```ts
export function ListServers(): $CancellablePromise<...> {
    return $Call.ByID(xxxx);
}
```

- [ ] **步骤 4：暂不 Commit（绑定是生成产物，下一任务改完前端再一起提交）**

---

## 任务 7：前端 api/index.ts 改用 v3 绑定

**文件：**
- 修改：`client/frontend/src/api/index.ts`

- [ ] **步骤 1：更新 api/index.ts 的绑定导入与 call 函数**

将 `client/frontend/src/api/index.ts` 顶部（第 1-96 行，从开头到 `call` 函数结束）替换为：

```ts
// 封装对 Wails v3 后端 App 服务的调用。
// v3 通过 wails3 generate bindings 自动生成 TypeScript 绑定，
// 前端直接 import 调用，不再依赖 window.go 注入。

import * as AppService from '../../bindings/github.com/kdc/frp-manager/client/app'

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

// 统一调用包装：v3 绑定方法返回 CancellablePromise，reject 时抛 Error。
// Wails reject 的值可能是字符串、对象或 Error，统一规范化成带 message 的 Error。
async function call<T>(fn: any, ...args: any[]): Promise<T> {
  if (!fn) {
    throw new Error('后端绑定未就绪')
  }
  try {
    return await fn(...args)
  } catch (e: any) {
    const msg = typeof e === 'string' ? e
      : (e?.message ?? e?.error ?? (typeof e === 'object' ? JSON.stringify(e) : String(e)))
    throw new Error(msg ?? '未知错误')
  }
}
```

- [ ] **步骤 2：更新 api 对象的方法调用**

将 `client/frontend/src/api/index.ts` 底部的 `export const api` 块（从 `export const api = {` 到文件末尾）替换为：

```ts
export const api = {
  // 服务器
  async listServers(): Promise<ServerInfo[]> {
    return (await call(AppService.ListServers)) ?? []
  },
  async addServer(input: AddServerInput): Promise<string> {
    return await call(AppService.AddServer, input)
  },
  async updateServerByID(id: string, input: AddServerInput): Promise<void> {
    await call(AppService.UpdateServerByID, id, input)
  },
  async deleteServer(id: string): Promise<void> {
    await call(AppService.DeleteServer, id)
  },
  async checkServerCapabilities(id: string): Promise<Capabilities> {
    return await call(AppService.CheckServerCapabilities, id)
  },

  // 映射
  async listTunnels(serverId?: string): Promise<TunnelInfo[]> {
    return (await call(AppService.ListTunnels, serverId ?? '')) ?? []
  },
  async addTunnel(input: AddTunnelInput): Promise<string> {
    return await call(AppService.AddTunnel, input)
  },
  async updateTunnelByID(id: string, input: AddTunnelInput): Promise<void> {
    await call(AppService.UpdateTunnelByID, id, input)
  },
  async deleteTunnel(id: string): Promise<void> {
    await call(AppService.DeleteTunnel, id)
  },

  // frpc
  async generateFrpcConfig(serverId: string): Promise<string> {
    return await call(AppService.GenerateFrpcConfig, serverId)
  },
  async startFrpc(serverId: string): Promise<void> {
    await call(AppService.StartFrpc, serverId)
  },
  async stopFrpc(serverId: string): Promise<void> {
    await call(AppService.StopFrpc, serverId)
  },
  async restartFrpc(serverId: string): Promise<void> {
    await call(AppService.RestartFrpc, serverId)
  },
  async isFrpcRunning(serverId: string): Promise<boolean> {
    return await call(AppService.IsFrpcRunning, serverId)
  },
}
```

**注意：** 移除了旧的 `backend()` 函数（v3 不再需要 window.go 注入）。方法名与后端 App 结构体的导出方法名一致（首字母大写），绑定生成会保留原名。

- [ ] **步骤 3：验证前端类型检查**

运行：
```bash
cd client/frontend
npx vue-tsc --noEmit
```
预期：可能因绑定类型细节报错（如返回类型不完全匹配），记录错误用于调整。若绑定方法名与预期不符，检查 `app.ts` 实际导出的函数名并调整 api/index.ts 的调用。

- [ ] **步骤 4：Commit**

```bash
git add client/frontend/src/api/index.ts client/frontend/bindings
git commit -m "refactor(client/frontend): api/index.ts 改用 v3 自动生成的 TypeScript 绑定"
```

---

## 任务 8：前端事件监听切换到 Events.On

**文件：**
- 修改：`client/frontend/src/App.vue`
- 修改：`client/frontend/src/stores/log.ts`

- [ ] **步骤 1：更新 App.vue 的事件导入与监听**

将 `client/frontend/src/App.vue` 的 `<script lang="ts" setup>` 块（第 2-26 行）替换为：

```ts
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Events } from '@wailsio/runtime'
import { useLogStore } from '@/stores/log'

const route = useRoute()
const router = useRouter()
const active = computed(() => route.path)
const logStore = useLogStore()

const menus = [
  { index: '/servers', label: '服务器', icon: 'Connection' },
  { index: '/tunnels', label: '映射', icon: 'Position' },
  { index: '/logs', label: '日志', icon: 'Document' },
  { index: '/settings', label: '设置', icon: 'Setting' },
]

onMounted(() => {
  // 绑定后端 app.Event.Emit 的 log:append 事件
  Events.On('log:append', (line: any) => {
    logStore.append({
      time: line.time ?? new Date().toISOString(),
      level: line.level ?? 'info',
      message: line.message ?? '',
      server_id: line.server_id,
    })
  })
})
```

- [ ] **步骤 2：更新 stores/log.ts 的事件绑定**

将 `client/frontend/src/stores/log.ts` 的 `bindEvents` 函数（约第 21-26 行）：
```ts
  // 绑定 Wails 事件 "log:append"，由后端 EventsEmit 推送。
  function bindEvents(runtime: any) {
    if (bound || !runtime?.EventsOn) return
    runtime.EventsOn('log:append', (line: LogLine) => append(line))
    bound = true
  }
```
替换为：
```ts
  // 绑定 Wails v3 事件 "log:append"，由后端 app.Event.Emit 推送。
  // App.vue 已通过 Events.On 绑定并调用 append，此处保留接口供兼容。
  function bindEvents(runtime: any) {
    if (bound) return
    bound = true
  }
```
**注意：** 实际事件绑定已在 App.vue 中通过 `Events.On` 完成，store 的 `bindEvents` 仅保留为空壳避免其他文件调用报错。若确认无其他调用方，可后续移除。

- [ ] **步骤 3：搜索确认无其他 EventsOn 残留**

运行：
```bash
cd client/frontend
grep -r "EventsOn\|wailsjs/runtime" src/
```
预期：无匹配（已全部替换为 `@wailsio/runtime` 的 `Events.On`）。

- [ ] **步骤 4：验证前端类型检查**

运行：
```bash
cd client/frontend
npx vue-tsc --noEmit
```
预期：PASS（或仅有非事件相关的类型告警）。

- [ ] **步骤 5：Commit**

```bash
git add client/frontend/src/App.vue client/frontend/src/stores/log.ts
git commit -m "refactor(client/frontend): 事件监听从 EventsOn 切换到 @wailsio/runtime Events.On"
```

---

## 任务 9：配置文件迁移（wails.json → Taskfile.yml + build/config.yml）

**文件：**
- 删除：`client/wails.json`
- 创建：`client/Taskfile.yml`
- 创建：`client/build/config.yml`
- 修改：`client/build/windows/info.json`
- 修改：`client/build/windows/installer/project.nsi`
- 创建：`client/build/windows/nsis/project.nsi`（从 wails3-ref 复制）

- [ ] **步骤 1：创建 Taskfile.yml**

创建 `client/Taskfile.yml`，内容：
```yaml
version: '3'

vars:
  APP_NAME: "frp-manager"
  BIN_DIR: "bin"
  PACKAGE_MANAGER: '{{.PACKAGE_MANAGER | default "npm"}}'
  VITE_PORT: '{{.WAILS_VITE_PORT | default 9245}}'
  GOOS: '{{.GOOS | default OS}}'

includes:
  common: ./build/Taskfile.yml
  windows: ./build/windows/Taskfile.yml
  darwin: ./build/darwin/Taskfile.yml
  linux: ./build/linux/Taskfile.yml
  ios: ./build/ios/Taskfile.yml
  android: ./build/android/Taskfile.yml

tasks:
  build:
    summary: Builds the application
    cmds:
      - task: "{{.GOOS}}:build"

  package:
    summary: Packages a production build of the application
    cmds:
      - task: "{{.GOOS}}:package"

  run:
    summary: Runs the application
    cmds:
      - task: "{{.GOOS}}:run"

  dev:
    summary: Runs the application in development mode
    cmds:
      - wails3 dev -config ./build/config.yml -port {{.VITE_PORT}}

  setup:docker:
    summary: Builds Docker image for cross-compilation (~800MB download)
    cmds:
      - task: common:setup:docker

  build:server:
    summary: Builds the application in server mode (no GUI, HTTP server only)
    cmds:
      - task: common:build:server

  run:server:
    summary: Runs the application in server mode
    cmds:
      - task: common:run:server

  build:docker:
    summary: Builds a Docker image for server mode deployment
    cmds:
      - task: common:build:docker

  run:docker:
    summary: Builds and runs the Docker image
    cmds:
      - task: common:run:docker
```

- [ ] **步骤 2：创建 build/config.yml**

创建 `client/build/config.yml`，内容：
```yaml
# Wails v3 项目配置
version: '3'

info:
  companyName: "kdc"
  productName: "FRP Manager"
  productIdentifier: "com.kdc.frp-manager"
  description: "基于 frp 的本地 GUI 内网穿透管理系统"
  copyright: "(c) 2026, kdc"
  comments: "FRP Manager 客户端"
  version: "0.1.0"

dev_mode:
  root_path: .
  log_level: warn
  debounce: 1000
  ignore:
    dir:
      - .git
      - node_modules
      - frontend
      - bin
    file:
      - .DS_Store
      - .gitignore
      - .gitkeep
      - "*_test.go"
    watched_extension:
      - "*.go"
      - "*.js"
      - "*.ts"
    git_ignore: true
  executes:
    - cmd: wails3 build DEV=true
      type: blocking
    - cmd: wails3 task common:dev:frontend
      type: background
    - cmd: wails3 task run
      type: primary
```

- [ ] **步骤 3：复制 build/Taskfile.yml 和平台 Taskfile**

从参照项目复制构建任务定义：
```bash
cd client
Copy-Item "C:\Users\15953\code\other\wails3-ref\build\Taskfile.yml" "build\Taskfile.yml"
Copy-Item "C:\Users\15953\code\other\wails3-ref\build\windows\Taskfile.yml" "build\windows\Taskfile.yml"
```

- [ ] **步骤 4：创建 nsis 目录并复制 v3 nsis 模板**

v3 的 nsis 安装脚本结构变化（从 `installer/` 移到 `nsis/`）：
```bash
cd client
New-Item -ItemType Directory -Force build\windows\nsis
Copy-Item "C:\Users\15953\code\other\wails3-ref\build\windows\nsis\project.nsi" "build\windows\nsis\project.nsi"
```
然后删除旧的 installer 目录：
```bash
Remove-Item -Recurse -Force build\windows\installer
```

- [ ] **步骤 5：更新 build/windows/info.json**

将 `client/build/windows/info.json` 全部内容替换为：
```json
{
	"fixed": {
		"file_version": "0.1.0"
	},
	"info": {
		"0000": {
			"ProductVersion": "0.1.0",
			"CompanyName": "kdc",
			"FileDescription": "FRP Manager",
			"LegalCopyright": "© 2026, kdc",
			"ProductName": "FRP Manager",
			"Comments": "基于 frp 的本地 GUI 内网穿透管理系统"
		}
	}
}
```

- [ ] **步骤 6：删除旧 wails.json**

运行：
```bash
cd client
Remove-Item wails.json
```

- [ ] **步骤 7：删除旧 build/darwin（v3 结构不同，本次 Windows 优先）**

运行：
```bash
cd client
Remove-Item -Recurse -Force build\darwin
```
**说明：** v3 的 darwin 构建目录结构变化（需 Assets.car 等），本次迁移以 Windows 验证为主。darwin 目录可在后续按 v3 模板重建。

- [ ] **步骤 8：验证 wails3 能识别项目**

运行：
```bash
cd client
wails3 task --summary
```
预期：列出 build/dev/run 等任务，无报错。

- [ ] **步骤 9：Commit**

```bash
cd client
git add Taskfile.yml build/config.yml build/Taskfile.yml build/windows/Taskfile.yml build/windows/nsis build/windows/info.json
git rm wails.json
git rm -r build/windows/installer build/darwin
git commit -m "chore(client): 配置文件迁移到 Wails v3（Taskfile.yml + build/config.yml + nsis）"
```

---

## 任务 10：端到端验证

**文件：**
- 验证：整个客户端

- [ ] **步骤 1：再次运行 Go 单测**

运行：
```bash
cd client
go test ./...
```
预期：全部 PASS。

- [ ] **步骤 2：Go 代码格式化**

运行：
```bash
cd client
gofmt -w main.go app.go app_test.go types.go
```

- [ ] **步骤 3：前端代码格式化**

运行：
```bash
cd client/frontend
npx prettier --write "src/**/*.{ts,vue}" vite.config.ts
```
若未安装 prettier，跳过（项目当前 package.json 无 prettier 依赖，可后续添加）。

- [ ] **步骤 4：启动开发模式验证 GUI**

运行：
```bash
cd client
wails3 task dev
```
预期：窗口启动显示 "FRP Manager"，左侧导航 服务器/映射/日志/设置 四个菜单可点击。

**手动验证清单：**
- [ ] 服务器页能加载列表（空列表无报错）
- [ ] 能添加服务器并显示
- [ ] 映射页能加载列表
- [ ] 日志页无报错（事件监听就绪）
- [ ] 设置页无报错

- [ ] **步骤 5：若 dev 模式启动失败，排查并修复**

常见问题：
- 绑定路径不匹配：检查 `frontend/bindings/github.com/kdc/frp-manager/client/app.ts` 是否存在，api/index.ts 的 import 路径是否正确
- 事件未推送：检查 app.go 的 `SetApplication` 是否在 `application.New` 后调用
- 资源 404：检查 `frontend/dist` 是否已 build，vite 插件路径是否正确

- [ ] **步骤 6：清理临时参照项目**

迁移验证完成后，删除参照项目：
```bash
Remove-Item -Recurse -Force C:\Users\15953\code\other\wails3-ref
```

- [ ] **步骤 7：最终 Commit（如有格式化或修复改动）**

```bash
cd client
git add .
git commit -m "style(client): Wails v3 迁移后格式化与端到端验证"
```
（若无非步骤 4 修复的改动则跳过）

---

## 自检结果

**规格覆盖度：** 逐项对照设计文档：
- §3.1 App 改造为 Service → 任务 2 ✓
- §3.2 EmitLog 改造 → 任务 2 步骤 6 ✓
- §3.3.1 api/index.ts → 任务 7 ✓
- §3.3.2 事件监听 → 任务 8 ✓
- §3.3.3 Vite 配置 → 任务 5 ✓
- §3.4 测试影响 → 任务 4 ✓
- §3.5 配置文件迁移 → 任务 9 ✓
- §3.6 依赖变更 → 任务 1（Go）+ 任务 5（前端）✓
- §4 迁移步骤顺序 → 任务 1-10 按序覆盖 ✓

**占位符扫描：** 无 TODO/待定，所有代码步骤含完整代码块。

**类型一致性：** `SetApplication`、`ServiceStartup`、`ServiceShutdown`、`EmitLog`、`InitForTest`、`Close` 方法名前后一致。前端 `AppService.ListServers` 等方法名与后端 App 导出方法一致。事件名 `log:append` 前后一致。
