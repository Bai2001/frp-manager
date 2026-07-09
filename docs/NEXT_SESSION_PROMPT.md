# FRP Manager 项目接手提示词

> 把本文件全部内容作为首条消息发给下一个会话的 AI 助手，它就能直接接手继续开发。

---

## 项目简介

FRP Manager 是基于 [frp](https://github.com/fatedier/frp) 的本地 GUI 内网穿透管理系统。服务端（frps + frp-server-agent）+ 客户端（Wails GUI + frpc）两部分，monorepo 结构。完整设计文档在 `docs/superpowers/specs/2026-07-07-frp-manager-design.md`。

**技术栈**：Go 1.26、Wails v2.12、Vue3 + TypeScript + Element Plus + Pinia + Vue Router、SQLite（modernc.org/sqlite 纯 Go 无 cgo）、chi 路由、frp v0.69.1（已内嵌）。

**仓库结构**：

```
frp-gui/
├─ go.work                 # Go workspace 聚合两端 module
├─ docs/superpowers/specs/ # 设计规格（必读）
├─ docs/superpowers/plans/ # 已完成的实现计划存档
├─ server/                 # 服务端 github.com/kdc/frp-manager/server
│  ├─ cmd/agent/           # agent 入口
│  ├─ internal/            # api/config/store/frps/portpool/domain/portprobe
│  ├─ migrations/          # SQLite DDL
│  └─ configs/             # agent.toml.example / frps.toml.example
└─ client/                 # 客户端 github.com/kdc/frp-manager/client
   ├─ main.go / app.go     # Wails 入口，App 暴露方法给前端
   ├─ internal/            # db/agent/frpc/config
   └─ frontend/            # Vue3 + TS + Element Plus
```

## 已完成（v0.1 核心闭环版 + frp 内嵌）

**v0.1 全部完成并验证通过**：

- 服务端 agent：`/api/health`、`/api/capabilities`、`/api/ports/{check,allocate,release}`、`/api/domains/{check,register,release}` 全部实现，Bearer token 鉴权，SQLite 存端口/域名占用
- 客户端 GUI：服务器增删改 + 检测能力、按协议动态创建映射（TCP/UDP/HTTP/HTTPS，在线校验端口/域名）、frpc 启停重启 + 状态轮询、实时日志（Wails 事件）、单实例锁
- frp **已内嵌**：客户端 import `github.com/fatedier/frp/client` 在进程内跑 frpc；服务端 agent import `github.com/fatedier/frp/server` 在进程内跑 frps。配置直接用官方 `v1.*Configurer` 对象构造，不走 TOML 文本往返
- 已验证：frpc 能真正连上 frps 建立隧道（login success、proxy start），Restart 竞态已修复（generation + done channel）

**已修复的 bug**：

- frpc Restart 后状态变停止（竞态：旧 goroutine 退出误删新 service）→ 用 generation 标记 + done channel 等待旧 service 退出
- 前端"创建失败: undefined"（Wails reject 字符串错误，`e.message` 取不到）→ `api/index.ts` 的 `call` 函数规范化错误为 Error

## 未完成任务（按设计文档第 21 节版本规划）

### v0.2：桌面体验版（下一阶段重点）

- [ ] **托盘运行**：v0.1 因 Wails v2.12 托盘 API 不稳定降级了，只保留单实例锁。需调研 Wails v2.12.0 的 `application.NewSystemTray` 或升级 Wails 版本实现托盘 + 右键菜单（显示主窗口/退出）
- [ ] **关闭最小化到托盘**：窗口关闭按钮不退出，最小化到托盘；托盘菜单提供"显示"/"退出"
- [ ] **开机自启**：Windows 用注册表 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`；Linux 用 `.desktop` autostart；macOS 用 `LaunchAgent`。Wails 可能有跨平台库
- [ ] **日志保留**：当前日志只存内存（pinia store 最多 2000 行，刷新丢失）。需持久化到文件（按天轮转，保留 N 天），设置页可配置保留天数
- [ ] **配置目录管理**：设置页展示/修改配置目录路径（当前 `internal/config.DefaultDir()` 写死用户配置目录）
- [ ] **导入导出**：导出 servers + tunnels 为 JSON 文件；导入恢复

### v0.3：服务端增强版

- [ ] **frps 启停 API 端点**：`frps.Manager` 已内嵌实现 Start/Stop/Restart，但 **agent 没暴露 HTTP 端点**（v0.1 设计文档第 4 节要求 agent "启动/停止/重启 frps"，但计划里归入 v0.3）。需加 `/api/frps/start`、`/api/frps/stop`、`/api/frps/restart`、`/api/frps/status`
- [ ] **frps 日志读取**：`/api/frps/logs` 端点，读取 frps 日志文件或捕获内嵌 frps 的 log 输出
- [ ] **端口池编辑**：服务端允许动态修改 allowPorts 范围（当前只能改 frps.toml 重启）
- [ ] **域名规则编辑**：服务端允许动态修改 allowed_root_domains / subdomain_host
- [ ] **frps 配置修复建议**：检测 frps.toml 常见问题并给建议
- [ ] **服务端一键检测**：客户端 GUI 按钮触发服务端全面健康检查

### v0.4：高级协议版

- [ ] **stcp / xtcp / tcpmux 协议支持**：frp 官方已支持，内嵌后直接用 `v1.STCPProxyConfig` / `v1.XTCPProxyConfig` / `v1.TCPMuxProxyConfig`。需扩展 `frpc.BuildProxy` 函数 + 前端 `TunnelFormDialog` 按协议动态字段
- [ ] **Basic Auth**：HTTP 映射加 `httpUser`/`httpPassword` 字段
- [ ] **HTTP Header 修改**：`requestHeaders`/`responseHeaders`/`hostHeaderRewrite`
- [ ] **健康检查**：`healthCheck.type = tcp/http`，配置超时/失败次数/间隔
- [ ] **负载均衡组**：`loadBalancer.group` + `groupKey`

### v1.0：稳定版

- [ ] **安装包发布**：`wails build` 生成 Windows/Linux/macOS 安装包，可能需 NSIS/MSI/dmg 打包
- [ ] **自动更新**：检查 GitHub Release 最新版，下载替换
- [ ] **多服务器同时运行**：当前 frpc.Manager 已支持 map[serverID]*Service 多实例，需验证 GUI 同时跑多个 frpc
- [ ] **错误诊断**：frpc 连接失败时给出具体原因（token 错/端口不通/域名冲突等）
- [ ] **frp 版本管理**：内嵌后版本固定，但可能需支持动态切换 frp 版本（降级回外部二进制模式？）
- [ ] **配置备份恢复**：导出完整配置快照，恢复到指定时间点

## 开发环境

- Windows + PowerShell，Go 1.26，Node 24，Wails v2.12
- Go 代理：`$env:GOPROXY="https://goproxy.cn,direct"`（网络问题时用）
- npm 镜像：`npm config set registry https://registry.npmmirror.com`
- 服务端启动：`cd server; go run ./cmd/agent -config configs/agent.local.toml`（本地测试配置，已 gitignore）
- 客户端启动：`cd client; wails dev`
- 测试：`cd server; go test ./...` 和 `cd client; go test ./...`
- 前端构建：`cd client/frontend; npm run build`
- **go.work 引用 ./server 和 ./client，在子目录跑 go 命令，勿在根目录跑 `go build ./...`**

## 已知风险/注意事项

1. **frps 内嵌启动时机**：当前 agent 启动时**不自动启动 frps**（v0.1 保持解耦）。本地测试时需手动通过 API 或临时在 main.go 加 `go frpsMgr.Start(context.Background())` 启动。v0.3 加了 frps 启停 API 后可由客户端控制
2. **frps.toml.example 的 TOML 键顺序问题**：顶层键（vhostHTTPPort 等）若写在 `[[allowPorts]]` 之后，TOML 规则会把它们归到 allowPorts 数组元素内。`frps.local.toml` 已修正顺序，但 `frps.toml.example` 未修（可顺手修）
3. **frp 日志集成**：frp 用全局 `pkg/util/log`，内嵌后日志走 stdout，未深度集成到 Wails 事件。v0.2 日志保留功能需用 `log.InitLogger` 重定向到自定义 writer
4. **Wails 托盘 API**：v2.12 的 `options.App` 无 `TrayMenu` 字段，`menu.TrayMenu` 类型存在但仅内部用。实现托盘可能需升级 Wails 或用平台原生 API（Windows 的 Shell_NotifyIcon）
5. **Wails reject 错误是字符串**：Go 方法返回 error 经 Wails 传到前端是字符串而非 Error 对象，前端 `catch(e)` 里用 `e.message` 会是 undefined。已用 `api/index.ts` 的 `call` 函数规范化，新增前端调用后端的方法时务必走 `call` 包装
6. **frpc Service.Run login 失败行为**：`LoginFailExit` 默认 true，连不上 frps 会退出。如需持续重试，在 `BuildClientConfig` 设 `common.LoginFailExit = lo.ToPtr(false)`

## 工作流约定（来自 CLAUDE.md）

- 完整功能完成之后才提交 git，不要频繁提交
- 命令产生的临时文件需清除
- 所有产物必须中文（输出文件、思考过程）
- 代码修改完成后用格式化工具（python 用 ruff，Go 用 gofmt，前端用 prettier）
- 涉及框架/软件配置问题必须用搜索工具
- 搜索代码逻辑优先用 mcp socraticode
- 简单改动不用 code-reviewer，复杂多页面改动才用
- 不会影响的多个任务可并发执行
- 复杂任务先写 md 计划文档再动手，简单任务直接实现

## 建议的下一步

从 **v0.2 桌面体验版** 开始。其中"托盘运行 + 关闭最小化"是用户感知最强的桌面体验改进。建议先调研 Wails 托盘方案（可能需升级 Wails 或用平台原生 API），再实现。如果托盘 API 仍不稳定，可先做"日志保留 + 配置目录管理 + 导入导出"这些纯应用层功能。

开始前建议先 `git log --oneline -20` 看完整提交历史，再读 `docs/superpowers/specs/` 和 `docs/superpowers/plans/` 下的文档了解上下文。
