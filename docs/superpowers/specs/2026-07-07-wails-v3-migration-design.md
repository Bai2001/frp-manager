# Wails v2 → v3 迁移设计

> 日期：2026-07-07
> 范围：客户端 `client/` 模块从 Wails v2.12.0 迁移到 Wails v3.0.0-alpha2.115
> 目标：完成核心迁移（应用结构、服务绑定、事件、前端调用），确保 v3 下 GUI + CRUD + frpc 启停 + 实时日志全部可用。托盘/最小化/开机自启留作 v0.2 后续任务。

---

## 1. 迁移动机

- **v0.2 托盘阻塞**：v2.12 的 `options.App` 未暴露托盘 API（仅内部 menumanager），导致托盘功能降级。v3 原生支持 `app.SystemTray.New()` + 菜单 + `AttachWindow`，是 v0.2 桌面体验的前置条件。
- **v3 架构改进**：应用创建/窗口创建/执行分离，服务模式解耦业务逻辑与运行时，多窗口支持，对象化 runtime 调用。
- **可共存**：v2/v3 import path 不同（`wails/v2` vs `wails/v3`），最坏情况可回退。

## 2. 影响范围盘点

### 2.1 Go 端变更

| 关注点 | v2 现状 | v3 新方式 |
|--------|---------|-----------|
| 应用启动 | `wails.Run(&options.App{})` 一把梭 | `application.New(opts)` → 创建窗口 → `app.Run()` |
| 资源服务 | `AssetServer.Assets: embed.FS` | `Assets: application.AssetOptions{Handler: application.AssetFileServerFS(assets)}` |
| 服务绑定 | `Bind: []interface{}{app}` | `Services: []application.Service{application.NewService(app)}` |
| 生命周期 | `OnStartup(ctx)` 存 ctx | `ServiceStartup(ctx, opts) error` + `ServiceShutdown() error` |
| 事件发射 | `runtime.EventsEmit(ctx, name, data)` | `app.Event.Emit(name, data)`（注入 `*application.App`） |
| 单实例锁 | `SingleInstanceLock.UniqueId` | `SingleInstance.UniqueID` + `OnSecondInstanceLaunch` |
| 窗口选项 | `options.App` 顶层 Width/Height | `WebviewWindowOptions` 独立结构体 |
| 托盘 | 未在 options 暴露 | `app.SystemTray.New()`（本次不实现，留 v0.2） |

### 2.2 前端变更

| 关注点 | v2 现状 | v3 新方式 |
|--------|---------|-----------|
| 调用后端 | `window.go.main.App.Xxx()` | `import { Xxx } from './bindings/<module>/app'` 自动生成 |
| 事件监听 | `import { EventsOn } from '../wailsjs/runtime/runtime'` | `import { Events } from '@wailsio/runtime'` → `Events.On(...)` |
| 运行时依赖 | `frontend/wailsjs/` 目录（dev 时注入） | npm 包 `@wailsio/runtime` + `frontend/bindings/` 自动生成 |
| Vite 配置 | 普通 vite 配置 | 需加 `wails` 插件：`wails("./bindings")` |

### 2.3 配置文件变更

| 关注点 | v2 | v3 |
|--------|----|----|
| 项目配置 | `wails.json` | `Taskfile.yml` + `build/config.yml` |
| 构建脚本 | `wails build` | `wails3 task build`（通过 Taskfile） |
| 开发命令 | `wails dev` | `wails3 task dev`（或 `wails3 dev -config ./build/config.yml`） |
| 绑定生成 | dev 时自动 | `wails3 generate bindings -ts` |

## 3. 架构设计

### 3.1 App 改造为 Service

当前 `App` 结构体持有 `ctx`/`database`/`repo`/`frpcMgr`，靠 `startup(ctx)` 拿 ctx 来发事件。v3 改造：

```go
type App struct {
    app      *application.App  // 注入，用于 Event.Emit
    ctx      context.Context   // ServiceStartup 传入
    database *sql.DB
    repo     *db.Repo
    frpcMgr  *frpc.Manager
}

// ServiceStartup 在应用启动时由 Wails v3 调用。
// 初始化 db/repo/frpcMgr，返回 error 可中断启动。
func (a *App) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
    a.ctx = ctx
    dbPath, err := config.DefaultDBPath()
    if err != nil { return fmt.Errorf("获取默认 DB 路径: %w", err) }
    database, err := db.Open(dbPath)
    if err != nil { return fmt.Errorf("打开客户端数据库: %w", err) }
    repo, err := db.NewRepo(database)
    if err != nil {
        database.Close()
        return fmt.Errorf("初始化 db repo: %w", err)
    }
    a.database = database
    a.repo = repo
    a.frpcMgr = frpc.NewManager()
    a.frpcMgr.SetLogCallback(func(serverID, line string) { /* 推日志事件 */ })
    return nil
}

// ServiceShutdown 在应用退出时由 Wails v3 调用。
func (a *App) ServiceShutdown() error {
    if a.database != nil { return a.database.Close() }
    return nil
}
```

**注入顺序问题**：Service 需要 `*application.App` 引用来发事件，但 `application.New(opts)` 需要 Service 实例。解法：
1. 先创建 App 实例（`NewApp()`）
2. `application.New(options)` 创建 app 实例并注册 service
3. 创建后调用 `appInstance.SetApplication(appInstance)` 把引用回填给 App 结构体

具体实现：
```go
func main() {
    appStruct := NewApp()
    app := application.New(application.Options{
        Name: "FRP Manager",
        Services: []application.Service{application.NewService(appStruct)},
        Assets: application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
        SingleInstance: &application.SingleInstanceOptions{
            UniqueID: "frp-manager-instance-lock",
            OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
                // 把已运行实例的窗口前置
            },
        },
    })
    appStruct.SetApplication(app)  // 回填 app 引用

    window := app.Window.NewWithOptions(application.WebviewWindowOptions{
        Title: "FRP Manager", Width: 1024, Height: 768,
        BackgroundColour: application.NewRGB(245, 247, 250),
        URL: "/",
    })
    window.Show()
    if err := app.Run(); err != nil { log.Fatal(err) }
}
```

### 3.2 EmitLog 改造

```go
func (a *App) EmitLog(level, message, serverID string) {
    if a.app == nil { return }  // 测试环境守卫
    a.app.Event.Emit("log:append", map[string]string{
        "time":      time.Now().UTC().Format(time.RFC3339),
        "level":     level,
        "message":   message,
        "server_id": serverID,
    })
}
```

### 3.3 前端改造

#### 3.3.1 api/index.ts

v3 绑定按 module name 分组，生成到 `frontend/bindings/<module-path>/`。我们的 module 是 `github.com/kdc/frp-manager/client`，绑定路径为 `frontend/bindings/github.com/kdc/frp-manager/client/`。

`call` 包装函数保留（Wails reject 错误规范化的逻辑不变）。`backend()` 函数从 `window.go.main.App` 改为直接 import 绑定：

```ts
// v3：直接 import 生成的绑定
import * as AppService from '../bindings/github.com/kdc/frp-manager/client/app'

async function call<T>(fn: any, ...args: any[]): Promise<T> {
  if (!fn) { throw new Error('后端绑定未就绪') }
  try { return await fn(...args) }
  catch (e: any) {
    const msg = typeof e === 'string' ? e
      : (e?.message ?? e?.error ?? (typeof e === 'object' ? JSON.stringify(e) : String(e)))
    throw new Error(msg ?? '未知错误')
  }
}

export const api = {
  async listServers(): Promise<ServerInfo[]> {
    return (await call(AppService.ListServers)) ?? []
  },
  async addServer(input: AddServerInput): Promise<string> {
    return await call(AppService.AddServer, input)
  },
  // ... 其余方法同理
}
```

#### 3.3.2 事件监听

`App.vue` 和 `stores/log.ts`：
```ts
import { Events } from '@wailsio/runtime'
Events.On('log:append', (line: LogLine) => { /* append */ })
```

#### 3.3.3 Vite 配置

```ts
import wails from '@wailsio/runtime/plugins/vite'
export default defineConfig({
  plugins: [vue(), wails('./bindings')],
  server: { host: '127.0.0.1', port: Number(process.env.WAILS_VITE_PORT) || 9245, strictPort: true },
})
```

### 3.4 测试影响

`app_test.go` 走 `InitForTest` 直接注入依赖，不经过 Wails 运行时：
- `InitForTest` 保持不变（直接设置 database/repo/frpcMgr）
- `EmitLog` 的 nil 守卫从 `if a.ctx == nil` 改为 `if a.app == nil`，测试中 `a.app` 为 nil 即跳过事件发射
- `Close()` 方法保留供测试 Cleanup 调用（v3 用 ServiceShutdown，但测试不经过生命周期）

### 3.5 配置文件迁移

**新增** `client/Taskfile.yml`（参照 wails3-ref 模板，APP_NAME 改为 `frp-manager`）。
**新增** `client/build/config.yml`（dev_mode 配置）。
**删除** `client/wails.json`。
**保留** `client/build/` 目录下的 windows/darwin 平台资源，但需按 v3 结构调整（参照 wails3-ref/build/windows）。

### 3.6 依赖变更

**go.mod**：
- 删除 `github.com/wailsapp/wails/v2 v2.12.0`
- 新增 `github.com/wailsapp/wails/v3 v3.0.0-alpha2.115`

**frontend/package.json**：
- 新增 `"@wailsio/runtime": "alpha"` 依赖
- `build` script 改为 `vue-tsc && vite build --mode production`（对齐 v3 模板）
- 删除 `frontend/wailsjs/` 旧目录（v3 用 `frontend/bindings/`）

## 4. 迁移步骤顺序

1. **Go 依赖切换**：改 go.mod，`go mod tidy` 拉 v3 依赖
2. **main.go 改造**：application.New + 窗口创建 + app.Run()
3. **app.go 改造**：ServiceStartup/ServiceShutdown + SetApplication + EmitLog 用 app.Event
4. **删除旧 wailsjs 目录**：`frontend/wailsjs/`
5. **前端依赖**：package.json 加 @wailsio/runtime，npm install
6. **Vite 配置**：加 wails 插件
7. **生成绑定**：`wails3 generate bindings -ts`
8. **api/index.ts 改造**：import 绑定替换 window.go
9. **App.vue + log.ts 改造**：Events.On 替换 EventsOn
10. **配置文件**：新增 Taskfile.yml + build/config.yml，删除 wails.json
11. **验证**：`wails3 task dev` 启动，测试 CRUD/启停/日志
12. **测试**：`go test ./...` 确保单测通过
13. **格式化**：gofmt + prettier

## 5. 风险与回退

- **v3 是 alpha 版**：可能有未知 bug。最坏情况 `git revert` 回退 v2，不影响 server 端（server 不依赖 wails）。
- **绑定生成失败**：若 module path 含特殊字符导致绑定路径异常，可调整 vite 的 wails 插件路径参数。
- **事件名兼容**：保持事件名 `log:append` 不变，前端只需改 import 方式。
- **SingleInstance 回调**：`OnSecondInstanceLaunch` 中前置窗口的 API 需确认（v3 用 `window.Show()` + `window.Focus()`）。

## 6. 不在本次范围

- 托盘运行 + 关闭最小化 + 开机自启（v0.2 后续，v3 API 已就绪）
- 日志保留 + 配置目录管理 + 导入导出（v0.2 后续）
- 服务端任何改动（server 不依赖 wails）
