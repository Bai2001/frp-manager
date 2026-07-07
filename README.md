# FRP Manager

基于 [frp](https://github.com/fatedier/frp) 的本地 GUI 内网穿透管理系统。由**服务端**（frps + frp-server-agent）和**客户端**（Wails GUI + frpc）两部分组成，所有管理操作在本地桌面端完成，不提供 Web 管理页面。

## 架构

```
┌─────────────────────────────────────┐
│ 本地客户端：FRP Manager Desktop      │
│ Wails + Vue3 + Go                   │
│ - 服务器连接管理                     │
│ - TCP/UDP/HTTP/HTTPS 映射管理        │
│ - 本地 frpc 配置生成                 │
│ - frpc 启停重启 + 实时日志           │
│ - 托盘运行 / 开机自启                │
└──────────────┬──────────────────────┘
               │ HTTP API
┌──────────────▼──────────────────────┐
│ 公网服务端：FRP Manager Server       │
│ - frps                              │
│ - frp-server-agent (Go)             │
│ - SQLite                            │
│ 能力：端口池/域名/frps 状态/配置约束 │
└──────────────┬──────────────────────┘
               │ frp 隧道
┌──────────────▼──────────────────────┐
│ 本地服务 (TCP/UDP/HTTP/HTTPS)        │
└─────────────────────────────────────┘
```

## 仓库结构

```
frp-gui/
├─ go.work                 # Go workspace，聚合两端 module
├─ docs/design.md          # 设计文档存档
├─ server/                 # 服务端（Go 单二进制 agent）
│  ├─ cmd/agent/           # agent 入口
│  ├─ internal/            # config/api/store/frps/portpool/domain
│  ├─ migrations/          # SQLite DDL
│  └─ configs/             # agent.toml / frps.toml 示例
└─ client/                 # 客户端（Wails v2 + Vue3 + Element Plus）
   ├─ main.go / app.go     # Wails 入口，暴露方法给前端
   ├─ internal/            # db/agent/frpc/config
   └─ frontend/            # Vue3 + TS + Vite
```

## 开发环境

- Go 1.26+
- Node 20+ / npm
- [Wails v2](https://wails.io) CLI：`go install github.com/wailsapp/wails/v2/cmd/wails@latest`

## 快速开始

### 服务端

```bash
cd server
go run ./cmd/agent -config configs/agent.toml.example
# 健康检查
curl http://127.0.0.1:7400/api/health
```

### 客户端

```bash
cd client
wails dev
```

## 协议支持（v0.1）

| 协议 | 远程端口 | 域名 | 典型场景 |
|------|---------|------|---------|
| TCP  | 是 | 否 | SSH/RDP/数据库 |
| UDP  | 是 | 否 | WireGuard/游戏/DNS |
| HTTP | 否 | 是 | Web 服务/API |
| HTTPS| 否 | 是 | HTTPS Web 服务 |

## 版本规划

- **v0.1**：服务端 agent + 四种协议映射闭环 + frpc 进程管理
- **v0.2**：托盘/开机自启/配置导入导出
- **v0.3**：服务端增强（frps 重启/日志/端口池编辑）
- **v0.4**：stcp/xtcp/tcpmux/Basic Auth
- **v1.0**：安装包发布/自动更新/多服务器

详细设计见 [docs/design.md](docs/design.md)。
