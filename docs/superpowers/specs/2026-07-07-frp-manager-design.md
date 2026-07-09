# FRP Manager 设计文档修正版

## 1. 项目定位

FRP Manager 是一个基于 frp 的本地 GUI 内网穿透管理系统。

项目由两个必选部分组成：

1. 服务端：frps + frp-server-agent
2. 客户端：Wails GUI + frpc

其中：

* 服务端必须部署。
* 服务端负责端口池、域名、frps 状态、配置约束、远程端口检测。
* 客户端负责本地 GUI 管理、映射创建、frpc 配置生成、frpc 进程管理。
* 不提供 Web 管理页面。
* 所有管理操作都在本地 Wails GUI 中完成。
* 底层穿透能力使用官方 frp。
* 第一版必须支持 tcp、udp、http、https 四类协议。

## 2. 总体架构

┌──────────────────────────────────────────────┐
│ 本地客户端：FRP Manager Desktop │
│ │
│ Wails + Vue3 + Go │
│ │
│ - 服务器连接管理 │
│ - TCP / UDP / HTTP / HTTPS 映射管理 │
│ - 本地 frpc 配置生成 │
│ - 本地 frpc 启动 / 停止 / 重启 │
│ - 实时日志展示 │
│ - 托盘运行 │
│ - 开机自启 │
└───────────────────────┬──────────────────────┘
 │
 │ HTTP API
 │
┌───────────────────────▼──────────────────────┐
│ 公网服务端：FRP Manager Server │
│ │
│ 必选组件： │
│ - frps │
│ - frp-server-agent │
│ │
│ 能力： │
│ - 端口池管理 │
│ - HTTP/HTTPS 域名校验 │
│ - frps 状态检测 │
│ - frps 配置约束 │
│ - 远程端口检测 │
│ - token 管理 │
└───────────────────────┬──────────────────────┘
 │
 │ frp 隧道
 │
┌───────────────────────▼──────────────────────┐
│ 本地服务 │
│ │
│ TCP: 127.0.0.1:3389 │
│ UDP: 127.0.0.1:51820 │
│ HTTP: 127.0.0.1:3000 │
│ HTTPS: 127.0.0.1:8443 │
└──────────────────────────────────────────────┘

## 3. 服务端设计

服务端是必选组件，不再是可选组件。

服务端包含：

frps
frp-server-agent
SQLite
systemd service

目录建议：

/opt/frp-manager-server/
├─ frps
├─ frp-server-agent
├─ config/
│ ├─ frps.toml
│ └─ agent.toml
├─ data/
│ └─ server.db
└─ logs/
 ├─ frps.log
 └─ agent.log

## 4. frp-server-agent 职责

frp-server-agent 是整个系统的服务端控制面。

它必须提供以下能力：

1. 服务端初始化
2. frps 状态检测
3. frps 启动 / 停止 / 重启
4. frps 配置读取
5. 端口池管理
6. 远程端口检测
7. TCP / UDP 端口分配
8. HTTP / HTTPS 域名规则校验
9. token 管理
10. 服务端系统信息查询
11. frps 日志读取

注意：

客户端不直接 SSH 到服务器执行命令。

客户端只通过 frp-server-agent API 管理服务端。

## 5. 支持协议

第一版必须支持：

tcp
udp
http
https

协议能力如下：

| 协议 | 是否需要远程端口 | 是否需要域名 | 典型场景 |
| --- | --- | --- | --- |
| TCP | 是 | 否 | SSH、RDP、数据库 |
| UDP | 是 | 否 | WireGuard、游戏、DNS |
| HTTP | 否 | 是 | Web 服务、API |
| HTTPS | 否 | 是 | HTTPS Web 服务 |

## 6. TCP 映射设计

TCP 映射需要远程端口。

### 示例

本地服务：127.0.0.1:3389
公网访问：1.2.3.4:20389

### frpc 配置

[[proxies]]
name = "rdp-20389"
type = "tcp"
localIP = "127.0.0.1"
localPort = 3389
remotePort = 20389

### 创建流程

用户创建 TCP 映射
 ↓
填写本地 IP、本地端口
 ↓
选择手动远程端口或自动分配
 ↓
客户端请求 server-agent 检测端口
 ↓
server-agent 分配远程端口
 ↓
客户端保存映射
 ↓
生成 frpc.toml
 ↓
启动或重启 frpc

## 7. UDP 映射设计

UDP 映射也需要远程端口。

### 示例

本地服务：127.0.0.1:51820
公网访问：1.2.3.4:25180

### frpc 配置

[[proxies]]
name = "wireguard-25180"
type = "udp"
localIP = "127.0.0.1"
localPort = 51820
remotePort = 25180

### 创建流程

用户创建 UDP 映射
 ↓
填写本地 IP、本地端口
 ↓
选择远程 UDP 端口
 ↓
server-agent 检测 UDP 端口
 ↓
保存映射
 ↓
生成 frpc.toml
 ↓
重启 frpc 生效

## 8. HTTP 映射设计

HTTP 映射不使用远程端口，而是使用域名。

frps 需要配置 HTTP vhost 端口。

### frps 配置

bindPort = 7000
vhostHTTPPort = 80

### HTTP 映射示例

本地服务：127.0.0.1:3000
公网访问：http://app.example.com

### frpc 配置

[[proxies]]
name = "web-app-http"
type = "http"
localIP = "127.0.0.1"
localPort = 3000
customDomains = ["app.example.com"]

### 创建流程

用户创建 HTTP 映射
 ↓
填写本地 IP、本地端口
 ↓
填写自定义域名
 ↓
客户端请求 server-agent 校验域名规则
 ↓
server-agent 检查域名是否已被占用
 ↓
客户端保存映射
 ↓
生成 frpc.toml
 ↓
重启 frpc 生效

### 服务端校验规则

server-agent 应校验：

1. 域名格式是否合法
2. 域名是否在允许的根域名范围内
3. 域名是否已被其他映射占用
4. 当前 frps 是否启用了 vhostHTTPPort

## 9. HTTPS 映射设计

HTTPS 映射同样使用域名。

frps 需要配置 HTTPS vhost 端口。

### frps 配置

bindPort = 7000
vhostHTTPSPort = 443

### HTTPS 映射示例

本地服务：127.0.0.1:8443
公网访问：https://secure.example.com

### frpc 配置

[[proxies]]
name = "secure-web-https"
type = "https"
localIP = "127.0.0.1"
localPort = 8443
customDomains = ["secure.example.com"]

### 创建流程

用户创建 HTTPS 映射
 ↓
填写本地 IP、本地端口
 ↓
填写自定义域名
 ↓
客户端请求 server-agent 校验域名
 ↓
server-agent 检查 HTTPS vhost 配置
 ↓
保存映射
 ↓
生成 frpc.toml
 ↓
重启 frpc 生效

## 10. HTTP / HTTPS 域名策略

为了避免用户随便绑定任意域名，服务端需要配置允许的域名规则。

### agent.toml

[domain]
allow\_custom\_domain = true
allowed\_root\_domains = [
 "example.com",
 "frp.example.com"
]

allow\_subdomain = true
subdomain\_host = "frp.example.com"

支持两种模式：

### 10.1 自定义域名模式

用户可以填写完整域名。

示例：

app.example.com
api.example.com
test.frp.example.com

要求：

域名必须属于 allowed\_root\_domains

### 10.2 子域名模式

用户只填写子域名前缀。

示例：

用户输入：demo
最终域名：demo.frp.example.com

frpc 配置可以使用：

[[proxies]]
name = "demo-http"
type = "http"
localIP = "127.0.0.1"
localPort = 3000
subdomain = "demo"

服务端 frps 配置：

subDomainHost = "frp.example.com"

## 11. frps 配置设计

服务端必须由 server-agent 管理或校验 frps.toml。

### frps.toml 推荐配置

bindPort = 7000

auth.method = "token"
auth.token = "your-frp-token"

allowPorts = [
 { start = 10000, end = 60000 }
]

vhostHTTPPort = 80
vhostHTTPSPort = 443

subDomainHost = "frp.example.com"

log.to = "/opt/frp-manager-server/logs/frps.log"
log.level = "info"
log.maxDays = 7

webServer.addr = "127.0.0.1"
webServer.port = 7500
webServer.user = "admin"
webServer.password = "strong-password"

说明：

bindPort:
 frpc 连接 frps 的控制端口。

allowPorts:
 限制 TCP / UDP 远程端口范围。

vhostHTTPPort:
 HTTP 映射入口端口，通常是 80。

vhostHTTPSPort:
 HTTPS 映射入口端口，通常是 443。

subDomainHost:
 子域名模式的根域名。

webServer:
 frp 官方 Dashboard，只监听 127.0.0.1，不对公网开放。

## 12. 客户端 frpc 配置生成

客户端根据本地 SQLite 中的服务器和映射生成 frpc.toml。

### 完整示例

serverAddr = "1.2.3.4"
serverPort = 7000

auth.method = "token"
auth.token = "your-frp-token"

[[proxies]]
name = "rdp-20389"
type = "tcp"
localIP = "127.0.0.1"
localPort = 3389
remotePort = 20389

[[proxies]]
name = "wireguard-25180"
type = "udp"
localIP = "127.0.0.1"
localPort = 51820
remotePort = 25180

[[proxies]]
name = "web-app-http"
type = "http"
localIP = "127.0.0.1"
localPort = 3000
customDomains = ["app.example.com"]

[[proxies]]
name = "secure-web-https"
type = "https"
localIP = "127.0.0.1"
localPort = 8443
customDomains = ["secure.example.com"]

## 13. 服务端 Agent API 设计

## 13.1 健康检查

GET /api/health

返回：

{
 "status": "ok",
 "version": "0.1.0"
}

## 13.2 查询服务端能力

客户端启动后应先查询服务端能力。

GET /api/capabilities
Authorization: Bearer <agent-token>

返回：

{
 "frps\_running": true,
 "frps\_version": "0.61.0",
 "bind\_port": 7000,
 "allow\_ports": [
 {
 "start": 10000,
 "end": 60000
 }
 ],
 "support\_tcp": true,
 "support\_udp": true,
 "support\_http": true,
 "support\_https": true,
 "vhost\_http\_port": 80,
 "vhost\_https\_port": 443,
 "subdomain\_host": "frp.example.com",
 "allowed\_root\_domains": [
 "example.com",
 "frp.example.com"
 ]
}

## 13.3 检查端口

用于 TCP / UDP。

GET /api/ports/check?protocol=tcp&port=20389
Authorization: Bearer <agent-token>

返回：

{
 "protocol": "tcp",
 "port": 20389,
 "available": true,
 "reason": "available"
}

## 13.4 自动分配端口

用于 TCP / UDP。

POST /api/ports/allocate
Authorization: Bearer <agent-token>
Content-Type: application/json

{
 "protocol": "tcp"
}

返回：

{
 "protocol": "tcp",
 "port": 20389
}

## 13.5 释放端口

POST /api/ports/release
Authorization: Bearer <agent-token>
Content-Type: application/json

{
 "protocol": "tcp",
 "port": 20389
}

返回：

{
 "success": true
}

## 13.6 检查域名

用于 HTTP / HTTPS。

POST /api/domains/check
Authorization: Bearer <agent-token>
Content-Type: application/json

{
 "protocol": "http",
 "domain": "app.example.com"
}

返回：

{
 "domain": "app.example.com",
 "available": true,
 "reason": "available"
}

## 13.7 注册域名

POST /api/domains/register
Authorization: Bearer <agent-token>
Content-Type: application/json

{
 "protocol": "http",
 "domain": "app.example.com",
 "tunnel\_id": "tunnel-001"
}

返回：

{
 "success": true
}

## 13.8 释放域名

POST /api/domains/release
Authorization: Bearer <agent-token>
Content-Type: application/json

{
 "protocol": "http",
 "domain": "app.example.com"
}

返回：

{
 "success": true
}

## 14. 服务端数据库设计

服务端必须保存端口和域名占用信息。

### port\_allocations

CREATE TABLE port\_allocations (
 id TEXT PRIMARY KEY,
 protocol TEXT NOT NULL,
 port INTEGER NOT NULL,
 tunnel\_id TEXT,
 client\_id TEXT,
 status TEXT NOT NULL DEFAULT 'allocated',
 created\_at DATETIME NOT NULL,
 updated\_at DATETIME NOT NULL,

 UNIQUE(protocol, port)
);

### domain\_allocations

CREATE TABLE domain\_allocations (
 id TEXT PRIMARY KEY,
 protocol TEXT NOT NULL,
 domain TEXT NOT NULL,
 tunnel\_id TEXT,
 client\_id TEXT,
 status TEXT NOT NULL DEFAULT 'allocated',
 created\_at DATETIME NOT NULL,
 updated\_at DATETIME NOT NULL,

 UNIQUE(protocol, domain)
);

### server\_settings

CREATE TABLE server\_settings (
 key TEXT PRIMARY KEY,
 value TEXT
);

## 15. 客户端数据库设计

### servers

CREATE TABLE servers (
 id TEXT PRIMARY KEY,
 name TEXT NOT NULL,
 host TEXT NOT NULL,
 frps\_port INTEGER NOT NULL DEFAULT 7000,
 frp\_token TEXT NOT NULL,
 agent\_url TEXT NOT NULL,
 agent\_token TEXT NOT NULL,
 is\_default INTEGER NOT NULL DEFAULT 0,
 remark TEXT,
 created\_at DATETIME NOT NULL,
 updated\_at DATETIME NOT NULL
);

注意：

现在 agent\_url 和 agent\_token 是必填项。

因为服务端 Agent 是必选组件。

### tunnels

CREATE TABLE tunnels (
 id TEXT PRIMARY KEY,
 server\_id TEXT NOT NULL,
 name TEXT NOT NULL,
 protocol TEXT NOT NULL,
 local\_ip TEXT NOT NULL DEFAULT '127.0.0.1',
 local\_port INTEGER NOT NULL,

 remote\_port INTEGER,
 custom\_domain TEXT,
 subdomain TEXT,

 enabled INTEGER NOT NULL DEFAULT 1,
 status TEXT NOT NULL DEFAULT 'stopped',
 last\_error TEXT,
 remark TEXT,

 created\_at DATETIME NOT NULL,
 updated\_at DATETIME NOT NULL,

 FOREIGN KEY (server\_id) REFERENCES servers(id)
);

字段说明：

remote\_port:
 tcp / udp 使用。

custom\_domain:
 http / https 使用。

subdomain:
 http / https 子域名模式使用。

## 16. 客户端页面调整

## 16.1 服务器页面

服务器页面必须填写：

服务器名称
公网 IP / 域名
frps 端口
frp token
agent 地址
agent token

新增按钮：

检测服务端能力
检测 HTTP 支持
检测 HTTPS 支持
检测端口池
检测域名配置

## 16.2 映射创建页面

映射创建页面按协议动态展示字段。

### TCP / UDP

显示字段：

映射名称
本地 IP
本地端口
远程端口
自动分配远程端口

### HTTP / HTTPS

显示字段：

映射名称
本地 IP
本地端口
域名模式：
 - 自定义域名
 - 子域名前缀

自定义域名：
 app.example.com

或子域名前缀：
 demo

## 17. 映射创建流程

## 17.1 TCP / UDP 创建流程

用户选择 tcp / udp
 ↓
填写本地地址和本地端口
 ↓
选择远程端口或自动分配
 ↓
客户端请求 server-agent 检查或分配端口
 ↓
server-agent 写入 port\_allocations
 ↓
客户端保存 tunnel
 ↓
生成 frpc.toml
 ↓
提示重启 frpc 生效

## 17.2 HTTP / HTTPS 创建流程

用户选择 http / https
 ↓
填写本地地址和本地端口
 ↓
填写 custom\_domain 或 subdomain
 ↓
客户端请求 server-agent 检查域名
 ↓
server-agent 校验域名规则
 ↓
server-agent 写入 domain\_allocations
 ↓
客户端保存 tunnel
 ↓
生成 frpc.toml
 ↓
提示重启 frpc 生效

## 18. 删除映射流程

删除映射时必须释放服务端资源。

### TCP / UDP

用户删除映射
 ↓
客户端请求 server-agent 释放端口
 ↓
删除本地 tunnel
 ↓
重新生成 frpc.toml
 ↓
如果 frpc 运行中，提示重启

### HTTP / HTTPS

用户删除映射
 ↓
客户端请求 server-agent 释放域名
 ↓
删除本地 tunnel
 ↓
重新生成 frpc.toml
 ↓
如果 frpc 运行中，提示重启

## 19. 服务端安装要求

服务端必须满足：

Linux 服务器
公网 IP
开放 frps bindPort，例如 7000
开放 TCP/UDP 端口池，例如 10000-60000
HTTP 映射需要开放 80
HTTPS 映射需要开放 443
开放 server-agent API 端口，例如 7400

推荐防火墙：

7000/tcp frps 控制端口
7400/tcp server-agent API
80/tcp HTTP vhost
443/tcp HTTPS vhost
10000-60000/tcp TCP 映射端口池
10000-60000/udp UDP 映射端口池

如果安全要求较高：

server-agent API 不直接暴露公网；
通过 VPN、内网、SSH Tunnel 或 Nginx HTTPS + IP 白名单访问。

## 20. MVP 范围修正

第一版 MVP 必须包含：

客户端：
- Wails GUI
- 添加服务端
- 检测 server-agent
- 检测服务端能力
- 创建 TCP 映射
- 创建 UDP 映射
- 创建 HTTP 映射
- 创建 HTTPS 映射
- 生成 frpc.toml
- 启动 / 停止 / 重启 frpc
- 实时日志
- 托盘运行

服务端：
- frp-server-agent
- frps 状态检测
- TCP / UDP 端口检测
- TCP / UDP 端口分配
- HTTP / HTTPS 域名检测
- HTTP / HTTPS 域名占用记录
- frps 配置能力读取

不进入第一版：

Web 管理页面
多用户系统
套餐系统
流量计费
自动安装服务端
自动更新
证书管理
Nginx 反代管理
多客户端协同管理

## 21. 版本规划修正

### v0.1：核心闭环版

目标：完整跑通服务端 + 客户端 + 四种协议。

必须支持：

server-agent
frps 状态检测
TCP 映射
UDP 映射
HTTP 映射
HTTPS 映射
frpc 配置生成
frpc 进程管理
日志展示

### v0.2：桌面体验版

托盘
开机自启
关闭最小化
日志保留
配置目录管理
导入导出

### v0.3：服务端增强版

服务端一键检测
frps 重启
frps 日志读取
端口池编辑
域名规则编辑
frps 配置修复建议

### v0.4：高级协议版

stcp
xtcp
tcpmux
Basic Auth
HTTP Header 修改
健康检查
负载均衡组

### v1.0：稳定版

安装包发布
自动更新
多服务器同时运行
错误诊断
frp 版本管理
配置备份恢复

## 22. 最终架构总结

最终系统由两部分组成：

服务端：
frps + frp-server-agent

客户端：
Wails GUI + frpc

职责边界：

服务端：
- 维护端口池
- 维护域名占用
- 暴露能力检测 API
- 管理 frps 状态
- 约束 TCP / UDP / HTTP / HTTPS 能力

客户端：
- 提供 GUI
- 创建映射
- 调用 server-agent 分配资源
- 生成 frpc.toml
- 启动和停止 frpc
- 展示状态与日志

第一版协议支持：

tcp
udp
http
https

这个版本已经不是单纯的 frpc GUI，而是：

带服务端 Agent 的 Wails 版 frp 管理系统。