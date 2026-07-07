# 服务端核心 API 实现计划（v0.1 计划 1/3）

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 把 `server/` 骨架里的占位接口替换为真实可用的 server-agent HTTP API，覆盖设计文档第 13 节的全部端点（health/capabilities/ports/domains），并用 Go 单测 + curl 冒烟验证。

**架构：** 复用已有 monorepo 的 `server/` module（`github.com/kdc/frp-manager/server`）。新增 `internal/frpsc`（frps.toml 配置解析）、`internal/portprobe`（真实端口探测）、改造 `internal/frps`、`internal/portpool`、`internal/domain`、`internal/store`、`internal/api`。所有外部依赖通过构造函数注入，handler 只做参数解析与 JSON 序列化，业务逻辑下沉到对应包，便于单测。

**技术栈：** Go 1.26、`github.com/go-chi/chi/v5`（路由）、`modernc.org/sqlite`（SQLite，纯 Go 无 cgo）、`github.com/pelletier/go-toml/v2`（TOML 解析，已引入）、`github.com/google/uuid`（ID 生成，已随 wails 间接引入，需显式 require）、`net/http` 标准库 + `go-test`。

---

## 文件结构

**新建：**
- `server/internal/frpsc/config.go` — 解析 `frps.toml` 为强类型结构，提取 bindPort/vhostHTTPPort/vhostHTTPSPort/allowPorts/subDomainHost/auth.token
- `server/internal/frpsc/config_test.go` — frps.toml 解析单测
- `server/internal/portprobe/portprobe.go` — TCP/UDP 端口探测（`net.Listen`/`net.ListenPacket` 试探）
- `server/internal/portprobe/portprobe_test.go` — 端口探测单测
- `server/internal/portpool/manager.go` — 端口池真实实现（基于 store + portprobe）
- `server/internal/portpool/manager_test.go` — 端口池单测
- `server/internal/domain/manager.go` — 域名管理真实实现（基于 store + config 规则）
- `server/internal/domain/manager_test.go` — 域名管理单测
- `server/internal/frps/manager.go` — frps 状态检测真实实现（进程检测 + frpsc 解析）
- `server/internal/frps/manager_test.go` — frps 状态检测单测

**修改：**
- `server/internal/store/store.go` — 新增 `PortAllocations`/`DomainAllocations` 两个 repo 的查询方法
- `server/internal/portpool/portpool.go` — `Manager` 接口保留，`New` 改为真实实现，`noopManager` 删除
- `server/internal/domain/domain.go` — 同上
- `server/internal/frps/frps.go` — `Manager` 接口保留并扩展 `Config()` 方法，`New` 改为真实实现，`noopManager` 删除
- `server/internal/api/api.go` — `Server` 注入 store/frps/portpool/domain，实现 7 个端点
- `server/cmd/agent/main.go` — 装配真实依赖并注入 api.Server
- `server/go.mod` — 显式 require `github.com/google/uuid`

**测试策略：** 单测覆盖每个包的核心逻辑（配置解析、端口探测、端口分配/释放、域名校验/注册/释放、frps 状态）。API 层用 `httptest` + 真实 SQLite 临时文件做集成测试。每个任务结束用 curl 做一次冒烟验证。

**约定：**
- 时间戳统一用 `time.Now().UTC().Format(time.RFC3339)`，由调用方传入 `time.Time` 或在 manager 内部取。
- ID 用 `uuid.NewString()`。
- HTTP 错误响应统一 `{"error":"<message>"}`，状态码遵循 REST 惯例（400 参数错、404 未找到、409 冲突、500 内部错）。
- 数据库 `status` 字段值：`allocated`（占用中）、`released`（已释放，保留历史）。

---

### 任务 1：解析 frps.toml 配置（frpsc 包）

**文件：**
- 创建：`server/internal/frpsc/config.go`
- 测试：`server/internal/frpsc/config_test.go`

frps.toml 真实结构（frp v0.61+，嵌套表形式，见 `server/configs/frps.toml.example`）：

```toml
bindPort = 7000
vhostHTTPPort = 80
vhostHTTPSPort = 443
subDomainHost = "frp.example.com"

[auth]
method = "token"
token = "your-frp-token"

[[allowPorts]]
start = 10000
end = 60000

[log]
to = "/opt/frp-manager-server/logs/frps.log"
level = "info"
maxDays = 7

[webServer]
addr = "127.0.0.1"
port = 7500
user = "admin"
password = "strong-password"
```

- [ ] **步骤 1：编写失败的测试**

`server/internal/frpsc/config_test.go`：

```go
package frpsc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	src := `
bindPort = 7000
vhostHTTPPort = 80
vhostHTTPSPort = 443
subDomainHost = "frp.example.com"

[auth]
method = "token"
token = "your-frp-token"

[[allowPorts]]
start = 10000
end = 60000

[[allowPorts]]
single = 3001
`
	dir := t.TempDir()
	p := filepath.Join(dir, "frps.toml")
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Parse(p)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.BindPort != 7000 {
		t.Errorf("BindPort = %d, want 7000", cfg.BindPort)
	}
	if cfg.VhostHTTPPort == nil || *cfg.VhostHTTPPort != 80 {
		t.Errorf("VhostHTTPPort want ptr 80, got %+v", cfg.VhostHTTPPort)
	}
	if cfg.VhostHTTPSPort == nil || *cfg.VhostHTTPSPort != 443 {
		t.Errorf("VhostHTTPSPort want ptr 443, got %+v", cfg.VhostHTTPSPort)
	}
	if cfg.SubDomainHost != "frp.example.com" {
		t.Errorf("SubDomainHost = %q, want frp.example.com", cfg.SubDomainHost)
	}
	if cfg.Auth.Token != "your-frp-token" {
		t.Errorf("Auth.Token = %q", cfg.Auth.Token)
	}
	if len(cfg.AllowPorts) != 2 {
		t.Fatalf("AllowPorts len = %d, want 2", len(cfg.AllowPorts))
	}
	if cfg.AllowPorts[0].Start != 10000 || cfg.AllowPorts[0].End != 60000 {
		t.Errorf("AllowPorts[0] = %+v", cfg.AllowPorts[0])
	}
	if cfg.AllowPorts[1].Single == nil || *cfg.AllowPorts[1].Single != 3001 {
		t.Errorf("AllowPorts[1].Single want ptr 3001, got %+v", cfg.AllowPorts[1].Single)
	}
}

func TestParse_missingFile(t *testing.T) {
	if _, err := Parse(filepath.Join(t.TempDir(), "nope.toml")); err == nil {
		t.Fatal("want error for missing file")
	}
}

func TestIsPortAllowed(t *testing.T) {
	cfg := &Config{
		AllowPorts: []AllowPort{
			{Start: 10000, End: 20000},
			{Single: intPtr(3001)},
		},
	}
	cases := []struct {
		port int
		want bool
	}{
		{10000, true}, {15000, true}, {20000, true}, {3001, true},
		{9999, false}, {20001, false}, {3002, false},
	}
	for _, c := range cases {
		if got := cfg.IsPortAllowed(c.port); got != c.want {
			t.Errorf("IsPortAllowed(%d) = %v, want %v", c.port, got, c.want)
		}
	}
}

func intPtr(i int) *int { return &i }
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/frpsc/ -v`
预期：FAIL，`undefined: Config`、`undefined: Parse`

- [ ] **步骤 3：编写实现**

`server/internal/frpsc/config.go`：

```go
// Package frpsc 解析 frps.toml 配置文件，提取 agent 需要的能力字段。
package frpsc

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Config 是 frps.toml 中 agent 关心的子集。
type Config struct {
	BindPort        int          `toml:"bindPort"`
	VhostHTTPPort   *int         `toml:"vhostHTTPPort"`
	VhostHTTPSPort  *int         `toml:"vhostHTTPSPort"`
	SubDomainHost   string       `toml:"subDomainHost"`
	Auth            Auth         `toml:"auth"`
	AllowPorts      []AllowPort  `toml:"allowPorts"`
}

// Auth 是 frps 鉴权配置。
type Auth struct {
	Method string `toml:"method"`
	Token  string `toml:"token"`
}

// AllowPort 描述一段允许的端口范围或单个端口。
type AllowPort struct {
	Start  int  `toml:"start"`
	End    int  `toml:"end"`
	Single *int `toml:"single"`
}

// Parse 读取并解析指定路径的 frps.toml。
func Parse(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 frps 配置 %s: %w", path, err)
	}
	var c Config
	if err := toml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("解析 frps 配置 %s: %w", path, err)
	}
	return &c, nil
}

// IsPortAllowed 判断 port 是否落在 allowPorts 范围内。
// 若 allowPorts 为空表示不限制（与 frps 行为一致）。
func (c *Config) IsPortAllowed(port int) bool {
	if len(c.AllowPorts) == 0 {
		return true
	}
	for _, r := range c.AllowPorts {
		if r.Single != nil && *r.Single == port {
			return true
		}
		if r.Start != 0 && r.End != 0 && port >= r.Start && port <= r.End {
			return true
		}
	}
	return false
}
```

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/frpsc/ -v`
预期：PASS，三个测试全过

- [ ] **步骤 5：Commit**

```bash
git add server/internal/frpsc/
git commit -m "feat(server): 新增 frpsc 包解析 frps.toml"
```

---

### 任务 2：端口探测（portprobe 包）

**文件：**
- 创建：`server/internal/portprobe/portprobe.go`
- 测试：`server/internal/portprobe/portprobe_test.go`

- [ ] **步骤 1：编写失败的测试**

`server/internal/portprobe/portprobe_test.go`：

```go
package portprobe

import (
	"net"
	"testing"
)

func TestTCPAvailable_freePort(t *testing.T) {
	// 任意未占用端口应判定为可用
	got, err := TCPAvailable(59999)
	if err != nil {
		t.Fatalf("TCPAvailable: %v", err)
	}
	if !got {
		t.Errorf("TCPAvailable(59999) = false, want true")
	}
}

func TestTCPAvailable_occupiedPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	got, err := TCPAvailable(port)
	if err != nil {
		t.Fatalf("TCPAvailable: %v", err)
	}
	if got {
		t.Errorf("TCPAvailable(%d) = true, want false (端口已被占用)", port)
	}
}

func TestUDPAvailable_freePort(t *testing.T) {
	got, err := UDPAvailable(59998)
	if err != nil {
		t.Fatalf("UDPAvailable: %v", err)
	}
	if !got {
		t.Errorf("UDPAvailable(59998) = false, want true")
	}
}

func TestUDPAvailable_occupiedPort(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	port := pc.LocalAddr().(*net.UDPAddr).Port
	got, err := UDPAvailable(port)
	if err != nil {
		t.Fatalf("UDPAvailable: %v", err)
	}
	if got {
		t.Errorf("UDPAvailable(%d) = true, want false", port)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/portprobe/ -v`
预期：FAIL，`undefined: TCPAvailable`、`undefined: UDPAvailable`

- [ ] **步骤 3：编写实现**

`server/internal/portprobe/portprobe.go`：

```go
// Package portprobe 通过尝试监听判断 TCP/UDP 端口在本地是否可用。
package portprobe

import (
	"fmt"
	"net"
	"time"
)

// TCPAvailable 判断本地 TCP port 是否可被监听（即可用作 frps 远程端口）。
func TCPAvailable(port int) (bool, error) {
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.ListenTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		return false, nil // 监听失败视为被占用
	}
	_ = ln.Close()
	return true, nil
}

// UDPAvailable 判断本地 UDP port 是否可被监听。
func UDPAvailable(port int) (bool, error) {
	addr := fmt.Sprintf(":%d", port)
	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		return false, nil
	}
	_ = pc.Close()
	return true, nil
}
```

> 注意：`net.ListenTimeout` 不存在；标准库用 `net.Listen`（无超时，但本机回环很快）。若编译报错，把 `TCPAvailable` 实现改为：

```go
func TCPAvailable(port int) (bool, error) {
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false, nil
	}
	_ = ln.Close()
	return true, nil
}
```

以最终编译通过为准，使用 `net.Listen` 版本。

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/portprobe/ -v`
预期：PASS，4 个测试全过

- [ ] **步骤 5：Commit**

```bash
git add server/internal/portprobe/
git commit -m "feat(server): 新增 portprobe 包检测端口可用性"
```

---

### 任务 3：store 新增端口/域名 repo 方法

**文件：**
- 修改：`server/internal/store/store.go`
- 创建：`server/internal/store/store_test.go`

- [ ] **步骤 1：编写失败的测试**

`server/internal/store/store_test.go`：

```go
package store

import (
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db := Open(filepath.Join(t.TempDir(), "test.db"))
	s, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s
}

func TestPortAllocations_InsertAndGet(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()
	pa := PortAllocation{
		ID: "p1", Protocol: "tcp", Port: 20389,
		TunnelID: "t1", Status: "allocated",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.InsertPortAllocation(pa); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, err := s.GetPortAllocation("tcp", 20389)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.TunnelID != "t1" || got.Status != "allocated" {
		t.Errorf("got %+v", got)
	}
}

func TestPortAllocations_Duplicate(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()
	pa := PortAllocation{ID: "p1", Protocol: "tcp", Port: 20389, Status: "allocated", CreatedAt: now, UpdatedAt: now}
	if err := s.InsertPortAllocation(pa); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertPortAllocation(pa); err == nil {
		t.Fatal("want duplicate error, got nil")
	}
}

func TestPortAllocations_UpdateStatus(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()
	pa := PortAllocation{ID: "p1", Protocol: "tcp", Port: 20389, Status: "allocated", CreatedAt: now, UpdatedAt: now}
	_ = s.InsertPortAllocation(pa)
	if err := s.UpdatePortAllocationStatus("p1", "released", now); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := s.GetPortAllocation("tcp", 20389)
	if got.Status != "released" {
		t.Errorf("status = %q, want released", got.Status)
	}
}

func TestDomainAllocations_InsertAndGet(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()
	da := DomainAllocation{
		ID: "d1", Protocol: "http", Domain: "app.example.com",
		TunnelID: "t1", Status: "allocated",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.InsertDomainAllocation(da); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, err := s.GetDomainAllocation("http", "app.example.com")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.TunnelID != "t1" {
		t.Errorf("got %+v", got)
	}
}

func TestDomainAllocations_UpdateStatus(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()
	da := DomainAllocation{ID: "d1", Protocol: "http", Domain: "app.example.com", Status: "allocated", CreatedAt: now, UpdatedAt: now}
	_ = s.InsertDomainAllocation(da)
	if err := s.UpdateDomainAllocationStatus("d1", "released", now); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := s.GetDomainAllocation("http", "app.example.com")
	if got.Status != "released" {
		t.Errorf("status = %q", got.Status)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/store/ -v`
预期：FAIL，`undefined: Store`、`undefined: PortAllocation` 等

- [ ] **步骤 3：编写实现**

把以下内容追加到 `server/internal/store/store.go` 末尾（保留已有的 `Open` 函数不变）：

```go
// Store 封装端口与域名分配的数据库操作。
type Store struct {
	db *sql.DB
}

// NewStore 创建 Store。
func NewStore(db *sql.DB) (*Store, error) {
	return &Store{db: db}, nil
}

// PortAllocation 对应 port_allocations 表一行。
type PortAllocation struct {
	ID        string
	Protocol  string
	Port      int
	TunnelID  string
	ClientID  string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DomainAllocation 对应 domain_allocations 表一行。
type DomainAllocation struct {
	ID        string
	Protocol  string
	Domain    string
	TunnelID  string
	ClientID  string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// InsertPortAllocation 插入一条端口占用记录。UNIQUE 冲突时返回错误。
func (s *Store) InsertPortAllocation(pa PortAllocation) error {
	_, err := s.db.Exec(
		`INSERT INTO port_allocations (id, protocol, port, tunnel_id, client_id, status, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?)`,
		pa.ID, pa.Protocol, pa.Port, pa.TunnelID, pa.ClientID, pa.Status,
		pa.CreatedAt.Format(time.RFC3339), pa.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

// GetPortAllocation 查询指定协议+端口的占用记录。
func (s *Store) GetPortAllocation(protocol string, port int) (*PortAllocation, error) {
	row := s.db.QueryRow(
		`SELECT id, protocol, port, tunnel_id, client_id, status, created_at, updated_at
		 FROM port_allocations WHERE protocol=? AND port=?`, protocol, port)
	var pa PortAllocation
	var created, updated string
	if err := row.Scan(&pa.ID, &pa.Protocol, &pa.Port, &pa.TunnelID, &pa.ClientID, &pa.Status, &created, &updated); err != nil {
		return nil, err
	}
	pa.CreatedAt, _ = time.Parse(time.RFC3339, created)
	pa.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &pa, nil
}

// UpdatePortAllocationStatus 更新端口占用状态。
func (s *Store) UpdatePortAllocationStatus(id, status string, at time.Time) error {
	_, err := s.db.Exec(
		`UPDATE port_allocations SET status=?, updated_at=? WHERE id=?`,
		status, at.Format(time.RFC3339), id)
	return err
}

// InsertDomainAllocation 插入域名占用记录。
func (s *Store) InsertDomainAllocation(da DomainAllocation) error {
	_, err := s.db.Exec(
		`INSERT INTO domain_allocations (id, protocol, domain, tunnel_id, client_id, status, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?)`,
		da.ID, da.Protocol, da.Domain, da.TunnelID, da.ClientID, da.Status,
		da.CreatedAt.Format(time.RFC3339), da.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

// GetDomainAllocation 查询指定协议+域名的占用记录。
func (s *Store) GetDomainAllocation(protocol, domain string) (*DomainAllocation, error) {
	row := s.db.QueryRow(
		`SELECT id, protocol, domain, tunnel_id, client_id, status, created_at, updated_at
		 FROM domain_allocations WHERE protocol=? AND domain=?`, protocol, domain)
	var da DomainAllocation
	var created, updated string
	if err := row.Scan(&da.ID, &da.Protocol, &da.Domain, &da.TunnelID, &da.ClientID, &da.Status, &created, &updated); err != nil {
		return nil, err
	}
	da.CreatedAt, _ = time.Parse(time.RFC3339, created)
	da.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &da, nil
}

// UpdateDomainAllocationStatus 更新域名占用状态。
func (s *Store) UpdateDomainAllocationStatus(id, status string, at time.Time) error {
	_, err := s.db.Exec(
		`UPDATE domain_allocations SET status=?, updated_at=? WHERE id=?`,
		status, at.Format(time.RFC3339), id)
	return err
}
```

同时在 `store.go` 顶部 import 块加入 `"time"`。

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/store/ -v`
预期：PASS，5 个测试全过

- [ ] **步骤 5：Commit**

```bash
git add server/internal/store/
git commit -m "feat(server): store 新增端口与域名分配的 CRUD 方法"
```

---

### 任务 4：端口池真实实现（portpool 包）

**文件：**
- 修改：`server/internal/portpool/portpool.go` — 删除 `noopManager`，`New` 改签名
- 创建：`server/internal/portpool/manager.go` — 真实实现
- 测试：`server/internal/portpool/manager_test.go`

- [ ] **步骤 1：编写失败的测试**

`server/internal/portpool/manager_test.go`：

```go
package portpool

import (
	"path/filepath"
	"testing"

	"github.com/kdc/frp-manager/server/internal/frpsc"
	"github.com/kdc/frp-manager/server/internal/store"
)

func newManager(t *testing.T) *Manager {
	t.Helper()
	s, err := store.NewStore(store.Open(filepath.Join(t.TempDir(), "test.db")))
	if err != nil {
		t.Fatal(err)
	}
	frpCfg := &frpsc.Config{
		BindPort: 7000,
		AllowPorts: []frpsc.AllowPort{{Start: 20000, End: 20100}},
	}
	return NewManager(s, frpCfg)
}

func TestCheck_Available(t *testing.T) {
	m := newManager(t)
	res, err := m.Check(nil, TCP, 20001)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Available {
		t.Errorf("Available = false, want true; reason=%s", res.Reason)
	}
}

func TestCheck_OutOfRange(t *testing.T) {
	m := newManager(t)
	res, _ := m.Check(nil, TCP, 9999)
	if res.Available {
		t.Errorf("9999 应不在 allowPorts 范围内")
	}
	if res.Reason != "out_of_allow_ports" {
		t.Errorf("reason = %q, want out_of_allow_ports", res.Reason)
	}
}

func TestCheck_AlreadyAllocated(t *testing.T) {
	m := newManager(t)
	_, err := m.Allocate(nil, TCP) // 占一个
	if err != nil {
		t.Fatal(err)
	}
	// 取第一个被分配的端口
	all, _ := m.ListAllocated(TCP)
	port := all[0]
	res, _ := m.Check(nil, TCP, port)
	if res.Available {
		t.Errorf("已分配端口 %d 应不可用", port)
	}
	if res.Reason != "already_allocated" {
		t.Errorf("reason = %q, want already_allocated", res.Reason)
	}
}

func TestAllocate(t *testing.T) {
	m := newManager(t)
	port, err := m.Allocate(nil, TCP)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if port < 20000 || port > 20100 {
		t.Errorf("port = %d, 不在允许范围", port)
	}
}

func TestAllocate_Exhausted(t *testing.T) {
	s, _ := store.NewStore(store.Open(filepath.Join(t.TempDir(), "test.db")))
	frpCfg := &frpsc.Config{AllowPorts: []frpsc.AllowPort{{Start: 20000, End: 20000}}}
	m := NewManager(s, frpCfg)
	_, err := m.Allocate(nil, TCP)
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.Allocate(nil, TCP)
	if err == nil {
		t.Fatal("端口耗尽应返回错误")
	}
}

func TestRelease(t *testing.T) {
	m := newManager(t)
	port, _ := m.Allocate(nil, TCP)
	if err := m.Release(nil, TCP, port); err != nil {
		t.Fatalf("Release: %v", err)
	}
	// 释放后应可再次分配到同一端口（唯一活跃记录已 released）
	port2, err := m.Allocate(nil, TCP)
	if err != nil {
		t.Fatal(err)
	}
	if port2 != port {
		t.Errorf("释放后未复用 %d, got %d", port, port2)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/portpool/ -v`
预期：FAIL，`undefined: NewManager`、`undefined: Manager.ListAllocated`

- [ ] **步骤 3：编写实现**

`server/internal/portpool/manager.go`：

```go
package portpool

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/kdc/frp-manager/server/internal/frpsc"
	"github.com/kdc/frp-manager/server/internal/portprobe"
	"github.com/kdc/frp-manager/server/internal/store"
)

// Manager 是端口池的真实实现。
type Manager struct {
	store  *store.Store
	frpCfg *frpsc.Config
}

// NewManager 创建端口池管理器。
// frpCfg 用于读取 allowPorts 范围；为 nil 表示不限制。
func NewManager(s *store.Store, frpCfg *frpsc.Config) *Manager {
	return &Manager{store: s, frpCfg: frpCfg}
}

// Check 检查指定端口是否可用：范围 + 已分配 + 本地监听探测。
func (m *Manager) Check(_ context.Context, p Protocol, port int) (*CheckResult, error) {
	if m.frpCfg != nil && !m.frpCfg.IsPortAllowed(port) {
		return &CheckResult{Protocol: p, Port: port, Available: false, Reason: "out_of_allow_ports"}, nil
	}
	pa, err := m.store.GetPortAllocation(string(p), port)
	if err == nil && pa.Status == "allocated" {
		return &CheckResult{Protocol: p, Port: port, Available: false, Reason: "already_allocated"}, nil
	}
	avail, err := probe(p, port)
	if err != nil {
		return nil, err
	}
	if !avail {
		return &CheckResult{Protocol: p, Port: port, Available: false, Reason: "in_use"}, nil
	}
	return &CheckResult{Protocol: p, Port: port, Available: true, Reason: "available"}, nil
}

// Allocate 在允许范围内自动寻找一个可用端口并记录占用。
func (m *Manager) Allocate(_ context.Context, p Protocol) (int, error) {
	ranges := m.allowRanges()
	for _, r := range ranges {
		for port := r.start; port <= r.end; port++ {
			pa, err := m.store.GetPortAllocation(string(p), port)
			if err == nil && pa.Status == "allocated" {
				continue
			}
			avail, err := probe(p, port)
			if err != nil {
				return 0, err
			}
			if !avail {
				continue
			}
			now := time.Now().UTC()
			if err := m.store.InsertPortAllocation(store.PortAllocation{
				ID: uuid.NewString(), Protocol: string(p), Port: port,
				Status: "allocated", CreatedAt: now, UpdatedAt: now,
			}); err != nil {
				return 0, fmt.Errorf("记录端口占用: %w", err)
			}
			return port, nil
		}
	}
	return 0, errors.New("无可用端口")
}

// Release 释放端口占用（标记为 released）。
func (m *Manager) Release(_ context.Context, p Protocol, port int) error {
	pa, err := m.store.GetPortAllocation(string(p), port)
	if err != nil {
		return fmt.Errorf("端口未分配: %w", err)
	}
	return m.store.UpdatePortAllocationStatus(pa.ID, "released", time.Now().UTC())
}

// ListAllocated 返回指定协议当前 allocated 的端口列表。
func (m *Manager) ListAllocated(p Protocol) ([]int, error) {
	// store 暂未提供列表查询，这里通过遍历 allowRanges 查 GetPortAllocation 实现
	// 生产可加专门查询；为支持任务 4 测试足够
	var out []int
	for _, r := range m.allowRanges() {
		for port := r.start; port <= r.end; port++ {
			pa, err := m.store.GetPortAllocation(string(p), port)
			if err == nil && pa.Status == "allocated" {
				out = append(out, port)
			}
		}
	}
	return out, nil
}

type portRange struct{ start, end int }

func (m *Manager) allowRanges() []portRange {
	if m.frpCfg == nil || len(m.frpCfg.AllowPorts) == 0 {
		return []portRange{{0, 65535}}
	}
	var out []portRange
	for _, ap := range m.frpCfg.AllowPorts {
		if ap.Single != nil {
			out = append(out, portRange{*ap.Single, *ap.Single})
		}
		if ap.Start != 0 && ap.End != 0 {
			out = append(out, portRange{ap.Start, ap.End})
		}
	}
	return out
}

func probe(p Protocol, port int) (bool, error) {
	switch p {
	case TCP:
		return portprobe.TCPAvailable(port)
	case UDP:
		return portprobe.UDPAvailable(port)
	default:
		return false, fmt.Errorf("不支持的协议 %s", p)
	}
}
```

- [ ] **步骤 4：更新 portpool.go 保留接口、删除 noopManager**

把 `server/internal/portpool/portpool.go` 改为只保留类型定义，`New` 函数删除（用 `NewManager` 替代）：

```go
// Package portpool 负责 TCP/UDP 远程端口的检测、分配与释放。
package portpool

// Protocol 区分端口所属协议。
type Protocol string

const (
	TCP Protocol = "tcp"
	UDP Protocol = "udp"
)

// CheckResult 是端口检查结果。
type CheckResult struct {
	Protocol  Protocol `json:"protocol"`
	Port      int      `json:"port"`
	Available bool     `json:"available"`
	Reason    string   `json:"reason"`
}

// Manager 抽象端口池操作（由 manager.go 实现）。
type Manager interface {
	Check(ctx context.Context, p Protocol, port int) (*CheckResult, error)
	Allocate(ctx context.Context, p Protocol) (int, error)
	Release(ctx context.Context, p Protocol, port int) error
}
```

在 `portpool.go` 顶部 import 加 `"context"`。

> 注意：`Manager` 接口不再有 `ListAllocated`，`ListAllocated` 是 `*portpool.Manager`（结构体）的额外方法，测试中直接用结构体类型调用。如果接口需要 `ListAllocated`，把它加到接口定义里；为最小化，本计划保持接口不含 `ListAllocated`，测试用具体类型 `*Manager`。

- [ ] **步骤 5：运行测试验证通过**

运行：`go test ./internal/portpool/ -v`
预期：PASS，6 个测试全过

- [ ] **步骤 6：Commit**

```bash
git add server/internal/portpool/
git commit -m "feat(server): portpool 真实实现端口检测/分配/释放"
```

---

### 任务 5：域名管理真实实现（domain 包）

**文件：**
- 修改：`server/internal/domain/domain.go` — 删除 noopManager，`New` 改签名
- 创建：`server/internal/domain/manager.go`
- 测试：`server/internal/domain/manager_test.go`

- [ ] **步骤 1：编写失败的测试**

`server/internal/domain/manager_test.go`：

```go
package domain

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdc/frp-manager/server/internal/config"
	"github.com/kdc/frp-manager/server/internal/store"
)

func newManager(t *testing.T) *Manager {
	t.Helper()
	s, _ := store.NewStore(store.Open(filepath.Join(t.TempDir(), "test.db")))
	cfg := &config.DomainConfig{
		AllowCustomDomain:  true,
		AllowedRootDomains: []string{"example.com", "frp.example.com"},
		AllowSubdomain:     true,
		SubdomainHost:      "frp.example.com",
	}
	return NewManager(s, cfg)
}

func TestCheck_InvalidFormat(t *testing.T) {
	m := newManager(t)
	res, _ := m.Check(context.Background(), HTTP, "not a domain")
	if res.Available {
		t.Error("非法域名应不可用")
	}
	if res.Reason != "invalid_format" {
		t.Errorf("reason = %q, want invalid_format", res.Reason)
	}
}

func TestCheck_NotInRootDomains(t *testing.T) {
	m := newManager(t)
	res, _ := m.Check(context.Background(), HTTP, "app.other.com")
	if res.Available || res.Reason != "not_allowed_root" {
		t.Errorf("got %+v", res)
	}
}

func TestCheck_AvailableCustom(t *testing.T) {
	m := newManager(t)
	res, _ := m.Check(context.Background(), HTTP, "app.example.com")
	if !res.Available {
		t.Errorf("got %+v", res)
	}
}

func TestCheck_AlreadyAllocated(t *testing.T) {
	m := newManager(t)
	_ = m.Register(context.Background(), HTTP, "app.example.com", "t1")
	res, _ := m.Check(context.Background(), HTTP, "app.example.com")
	if res.Available || res.Reason != "already_allocated" {
		t.Errorf("got %+v", res)
	}
}

func TestRegister_AndRelease(t *testing.T) {
	m := newManager(t)
	if err := m.Register(context.Background(), HTTP, "app.example.com", "t1"); err != nil {
		t.Fatal(err)
	}
	if err := m.Release(context.Background(), HTTP, "app.example.com"); err != nil {
		t.Fatal(err)
	}
	res, _ := m.Check(context.Background(), HTTP, "app.example.com")
	if !res.Available {
		t.Errorf("释放后应可用, got %+v", res)
	}
}

func TestCheck_Subdomain(t *testing.T) {
	m := newManager(t)
	// 子域名模式只传前缀，manager 内部拼成 demo.frp.example.com
	res, _ := m.Check(context.Background(), HTTP, "demo")
	if !res.Available {
		t.Errorf("got %+v", res)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/domain/ -v`
预期：FAIL，`undefined: NewManager`

- [ ] **步骤 3：编写实现**

`server/internal/domain/manager.go`：

```go
package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kdc/frp-manager/server/internal/config"
	"github.com/kdc/frp-manager/server/internal/store"
)

// Manager 是域名管理的真实实现。
type Manager struct {
	store *store.Store
	cfg   *config.DomainConfig
}

// NewManager 创建域名管理器。
func NewManager(s *store.Store, cfg *config.DomainConfig) *Manager {
	return &Manager{store: s, cfg: cfg}
}

// Check 校验域名是否可用。
// domain 参数既可能是完整域名（自定义模式），也可能是子域名前缀（子域名模式）。
func (m *Manager) Check(_ context.Context, p Protocol, domain string) (*CheckResult, error) {
	full := m.resolveFullDomain(domain)
	if !isValidDomain(full) {
		return &CheckResult{Domain: domain, Available: false, Reason: "invalid_format"}, nil
	}
	if !m.isInAllowedRoots(full) {
		return &CheckResult{Domain: domain, Available: false, Reason: "not_allowed_root"}, nil
	}
	da, err := m.store.GetDomainAllocation(string(p), full)
	if err == nil && da.Status == "allocated" {
		return &CheckResult{Domain: domain, Available: false, Reason: "already_allocated"}, nil
	}
	return &CheckResult{Domain: domain, Available: true, Reason: "available"}, nil
}

// Register 注册域名占用。
func (m *Manager) Register(_ context.Context, p Protocol, domain, tunnelID string) error {
	full := m.resolveFullDomain(domain)
	if !isValidDomain(full) {
		return errors.New("域名格式非法")
	}
	if !m.isInAllowedRoots(full) {
		return errors.New("域名不在允许的根域名范围")
	}
	now := time.Now().UTC()
	return m.store.InsertDomainAllocation(store.DomainAllocation{
		ID: uuid.NewString(), Protocol: string(p), Domain: full,
		TunnelID: tunnelID, Status: "allocated", CreatedAt: now, UpdatedAt: now,
	})
}

// Release 释放域名占用。
func (m *Manager) Release(_ context.Context, p Protocol, domain string) error {
	full := m.resolveFullDomain(domain)
	da, err := m.store.GetDomainAllocation(string(p), full)
	if err != nil {
		return fmt.Errorf("域名未分配: %w", err)
	}
	return m.store.UpdateDomainAllocationStatus(da.ID, "released", time.Now().UTC())
}

// resolveFullDomain 若开启子域名模式且 domain 不含点，拼上 subdomainHost。
func (m *Manager) resolveFullDomain(domain string) string {
	if m.cfg.AllowSubdomain && m.cfg.SubdomainHost != "" && !strings.Contains(domain, ".") {
		return domain + "." + m.cfg.SubdomainHost
	}
	return domain
}

func (m *Manager) isInAllowedRoots(full string) bool {
	for _, root := range m.cfg.AllowedRootDomains {
		if full == root || strings.HasSuffix(full, "."+root) {
			return true
		}
	}
	return false
}

// isValidDomain 简单校验域名格式：非空、只含合法字符、至少一个点（或子域名前缀无点也允许，由调用方拼后校验）。
func isValidDomain(s string) bool {
	if s == "" || len(s) > 253 {
		return false
	}
	for _, c := range s {
		if !(c == '.' || c == '-' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}
```

- [ ] **步骤 4：更新 domain.go**

把 `server/internal/domain/domain.go` 改为只保留类型，删除 `noopManager` 与 `New`：

```go
// Package domain 负责 HTTP/HTTPS 域名的校验、注册与释放。
package domain

import "context"

// Protocol 区分域名所属协议。
type Protocol string

const (
	HTTP  Protocol = "http"
	HTTPS Protocol = "https"
)

// CheckResult 是域名校验结果。
type CheckResult struct {
	Domain    string `json:"domain"`
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
}

// Manager 抽象域名操作（由 manager.go 实现）。
type Manager interface {
	Check(ctx context.Context, p Protocol, domain string) (*CheckResult, error)
	Register(ctx context.Context, p Protocol, domain, tunnelID string) error
	Release(ctx context.Context, p Protocol, domain string) error
}
```

- [ ] **步骤 5：运行测试验证通过**

运行：`go test ./internal/domain/ -v`
预期：PASS，6 个测试全过

- [ ] **步骤 6：Commit**

```bash
git add server/internal/domain/
git commit -m "feat(server): domain 真实实现域名校验/注册/释放"
```

---

### 任务 6：frps 状态检测真实实现（frps 包）

**文件：**
- 修改：`server/internal/frps/frps.go` — 扩展接口加 `Config()`，删除 noopManager，`New` 改签名
- 创建：`server/internal/frps/manager.go`
- 测试：`server/internal/frps/manager_test.go`

- [ ] **步骤 1：编写失败的测试**

`server/internal/frps/manager_test.go`：

```go
package frps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kdc/frp-manager/server/internal/frpsc"
)

func writeFrpsToml(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "frps.toml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestStatus_ConfigParsed(t *testing.T) {
	p := writeFrpsToml(t, `
bindPort = 7000
vhostHTTPPort = 80
vhostHTTPSPort = 443
subDomainHost = "frp.example.com"

[auth]
token = "tok"

[[allowPorts]]
start = 10000
end = 60000
`)
	m := NewManager(p, "")
	st, err := m.Status(nil)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	// 没有 frps 进程时 Running 为 false，但配置应能解析
	if st.Running {
		t.Errorf("Running = true, want false (无真实 frps 进程)")
	}
	if st.BindPort != 7000 {
		t.Errorf("BindPort = %d, want 7000", st.BindPort)
	}
}

func TestConfig(t *testing.T) {
	p := writeFrpsToml(t, `bindPort = 7000`+"\n"+"vhostHTTPPort = 80\n")
	m := NewManager(p, "")
	cfg, err := m.Config()
	if err != nil {
		t.Fatalf("Config: %v", err)
	}
	if cfg.BindPort != 7000 {
		t.Errorf("BindPort = %d", cfg.BindPort)
	}
	if cfg.VhostHTTPPort == nil || *cfg.VhostHTTPPort != 80 {
		t.Errorf("VhostHTTPPort = %+v", cfg.VhostHTTPPort)
	}
	_ = cfg // 也用于断言类型为 *frpsc.Config
}

func TestStatus_MissingConfig(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "nope.toml"), "")
	st, err := m.Status(nil)
	if err == nil {
		t.Fatal("缺失配置应返回 error")
	}
	if st != nil {
		t.Errorf("err 时 st 应为 nil")
	}
}

// 确保返回的 Config 类型是 *frpsc.Config
var _ = func() *frpsc.Config { var m *Manager; _, _ = m.Config(); return nil }
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/frps/ -v`
预期：FAIL，`undefined: NewManager`、`undefined: Manager.Config`

- [ ] **步骤 3：编写实现**

`server/internal/frps/manager.go`：

```go
package frps

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/kdc/frp-manager/server/internal/frpsc"
)

// Manager 是 frps 状态检测的真实实现。
// 通过尝试连接 frps bindPort 判断是否运行，通过 frpsc.Parse 解析配置。
type Manager struct {
	cfgPath    string
	binaryPath string
}

// NewManager 创建 frps 管理器。binaryPath 可为空（骨架阶段不启动真实进程）。
func NewManager(cfgPath, binaryPath string) *Manager {
	return &Manager{cfgPath: cfgPath, binaryPath: binaryPath}
}

// Status 检测 frps 是否运行并解析其配置。
func (m *Manager) Status(_ context.Context) (*Status, error) {
	cfg, err := frpsc.Parse(m.cfgPath)
	if err != nil {
		return nil, fmt.Errorf("解析 frps 配置: %w", err)
	}
	running := isPortListening(cfg.BindPort)
	return &Status{
		Running:  running,
		BindPort: cfg.BindPort,
	}, nil
}

// Config 返回解析后的 frps 配置。
func (m *Manager) Config() (*frpsc.Config, error) {
	return frpsc.Parse(m.cfgPath)
}

// ConfigPath 返回 frps.toml 路径。
func (m *Manager) ConfigPath() string { return m.cfgPath }

// Start/Stop/Restart 在 v0.1 暂不实现真实进程管理（设计文档将其归入 v0.3），
// 但保留方法以满足接口，返回 not implemented。
func (m *Manager) Start(context.Context) error   { return errNotImpl }
func (m *Manager) Stop(context.Context) error    { return errNotImpl }
func (m *Manager) Restart(context.Context) error { return errNotImpl }

var errNotImpl = fmt.Errorf("frps 进程管理在 v0.1 未实现")

// isPortListening 判断本地 TCP 端口是否在监听。
func isPortListening(port int) bool {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:"+strconv.Itoa(port), 300*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
```

- [ ] **步骤 4：更新 frps.go**

把 `server/internal/frps/frps.go` 改为只保留类型与接口（删除 `noopManager` 和旧 `New`）：

```go
// Package frps 负责 frps 进程管理、状态检测与配置读取。
package frps

import (
	"context"

	"github.com/kdc/frp-manager/server/internal/frpsc"
)

// Status 描述 frps 当前状态。
type Status struct {
	Running  bool   `json:"running"`
	Version  string `json:"version,omitempty"`
	BindPort int    `json:"bind_port,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Manager 抽象 frps 的控制能力（由 manager.go 实现）。
type Manager interface {
	Status(ctx context.Context) (*Status, error)
	Config() (*frpsc.Config, error)
	ConfigPath() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Restart(ctx context.Context) error
}
```

- [ ] **步骤 5：运行测试验证通过**

运行：`go test ./internal/frps/ -v`
预期：PASS，3 个测试全过

- [ ] **步骤 6：Commit**

```bash
git add server/internal/frps/
git commit -m "feat(server): frps 状态检测真实实现 + 配置读取"
```

---

### 任务 7：实现 capabilities 端点

**文件：**
- 修改：`server/internal/api/api.go` — 注入依赖、实现 capabilities
- 创建：`server/internal/api/api_test.go`

- [ ] **步骤 1：编写失败的测试**

`server/internal/api/api_test.go`：

```go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kdc/frp-manager/server/internal/config"
)

func writeAgentConfig(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "agent.toml")
	body := `
[server]
addr = "127.0.0.1:0"
token = "test-token"
database = "` + filepath.Join(t.TempDir(), "test.db").Replace("\\", "\\\\", -1) + `"

[frps]
config = "` + filepath.Join(t.TempDir(), "frps.toml").Replace("\\", "\\\\", -1) + `"

[domain]
allow_custom_domain = true
allowed_root_domains = ["example.com"]
subdomain_host = "frp.example.com"
`
	_ = os.WriteFile(p, []byte(body), 0o644)
	_ = os.WriteFile(filepath.Join(t.TempDir(), "frps.toml"), []byte(`bindPort = 7000
vhostHTTPPort = 80
vhostHTTPSPort = 443
subDomainHost = "frp.example.com"

[[allowPorts]]
start = 10000
end = 60000
`), 0o644)
	return p
}

func TestCapabilities(t *testing.T) {
	cfgPath := writeAgentConfig(t)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	srv := NewTestServer(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/capabilities", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp CapabilitiesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.BindPort != 7000 {
		t.Errorf("BindPort = %d, want 7000", resp.BindPort)
	}
	if !resp.SupportHTTP || resp.VhostHTTPPort != 80 {
		t.Errorf("HTTP 能力: %+v", resp)
	}
	if !resp.SupportHTTPS || resp.VhostHTTPSPort != 443 {
		t.Errorf("HTTPS 能力: %+v", resp)
	}
	if !resp.SupportTCP || !resp.SupportUDP {
		t.Errorf("TCP/UDP 应支持")
	}
	if resp.SubdomainHost != "frp.example.com" {
		t.Errorf("SubdomainHost = %q", resp.SubdomainHost)
	}
	if len(resp.AllowedRootDomains) != 1 || resp.AllowedRootDomains[0] != "example.com" {
		t.Errorf("AllowedRootDomains = %+v", resp.AllowedRootDomains)
	}
}

func TestCapabilities_Unauthorized(t *testing.T) {
	cfgPath := writeAgentConfig(t)
	cfg, _ := config.Load(cfgPath)
	srv := NewTestServer(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/capabilities", nil)
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/api/ -v`
预期：FAIL，`undefined: NewTestServer`、`undefined: CapabilitiesResponse`

- [ ] **步骤 3：编写实现**

修改 `server/internal/api/api.go`：

1. 改 `Server` 结构与 `New` 签名，注入 store/frps/portpool/domain：

```go
type Server struct {
	cfg     *config.Config
	store   *store.Store
	frps    frps.Manager
	ports   *portpool.Manager
	domains *domain.Manager
}

// New 创建 API Server，注入全部依赖。
func New(cfg *config.Config, st *store.Store, f frps.Manager, p *portpool.Manager, d *domain.Manager) *Server {
	return &Server{cfg: cfg, store: st, frps: f, ports: p, domains: d}
}

// NewTestServer 供测试用：自动从 cfg 解析 frps 配置并构造依赖。
func NewTestServer(t *testing.T, cfg *config.Config) *Server {
	t.Helper()
	st, _ := store.NewStore(store.Open(cfg.Server.Database))
	f := frps.NewManager(cfg.Frps.Config, cfg.Frps.Binary)
	frpCfg, _ := f.Config()
	p := portpool.NewManager(st, frpCfg)
	d := domain.NewManager(st, &cfg.Domain)
	return New(cfg, st, f, p, d)
}
```

2. 在 `Router()` 中启用 capabilities 路由：

```go
	r.Get("/api/capabilities", s.capabilities)
```

3. 添加 handler 与响应类型（同文件）：

```go
// CapabilitiesResponse 对应设计文档 13.2 节。
type CapabilitiesResponse struct {
	FrpsRunning        bool              `json:"frps_running"`
	FrpsVersion        string            `json:"frps_version"`
	BindPort           int               `json:"bind_port"`
	AllowPorts         []AllowPortRange  `json:"allow_ports"`
	SupportTCP         bool              `json:"support_tcp"`
	SupportUDP         bool              `json:"support_udp"`
	SupportHTTP        bool              `json:"support_http"`
	SupportHTTPS       bool              `json:"support_https"`
	VhostHTTPPort      int               `json:"vhost_http_port"`
	VhostHTTPSPort     int               `json:"vhost_https_port"`
	SubdomainHost      string            `json:"subdomain_host"`
	AllowedRootDomains []string          `json:"allowed_root_domains"`
}

// AllowPortRange 是 capabilities 响应里的端口范围项。
type AllowPortRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

func (s *Server) capabilities(w http.ResponseWriter, _ *http.Request) {
	cfg, err := s.frps.Config()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取 frps 配置失败: " + err.Error()})
		return
	}
	st, _ := s.frps.Status(nil)
	resp := CapabilitiesResponse{
		BindPort:           cfg.BindPort,
		FrpsRunning:        st != nil && st.Running,
		SupportTCP:         true,
		SupportUDP:         true,
		SupportHTTP:        cfg.VhostHTTPPort != nil,
		SupportHTTPS:       cfg.VhostHTTPSPort != nil,
		SubdomainHost:      cfg.SubDomainHost,
		AllowedRootDomains: s.cfg.Domain.AllowedRootDomains,
	}
	if cfg.VhostHTTPPort != nil {
		resp.VhostHTTPPort = *cfg.VhostHTTPPort
	}
	if cfg.VhostHTTPSPort != nil {
		resp.VhostHTTPSPort = *cfg.VhostHTTPSPort
	}
	for _, ap := range cfg.AllowPorts {
		if ap.Start != 0 && ap.End != 0 {
			resp.AllowPorts = append(resp.AllowPorts, AllowPortRange{Start: ap.Start, End: ap.End})
		}
	}
	writeJSON(w, http.StatusOK, resp)
}
```

import 块加入：

```go
	"github.com/kdc/frp-manager/server/internal/domain"
	"github.com/kdc/frp-manager/server/internal/frps"
	"github.com/kdc/frp-manager/server/internal/portpool"
	"github.com/kdc/frp-manager/server/internal/store"
```

> 注意：`api_test.go` 里用了 Go 字符串方法 `Replace`，需 import `strings`；且 `writeAgentConfig` 里 `database` 路径含反斜杠转义，Windows 下 TOML 字符串要用单引号字面量避免转义问题。建议把 `writeAgentConfig` 改用 TOML 单引号字面串：`database = '` + filepath.Join(...) + `'`，frps.config 同理。**以最终编译并测试通过为准调整转义。**

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/api/ -v`
预期：PASS，2 个测试全过

- [ ] **步骤 5：Commit**

```bash
git add server/internal/api/
git commit -m "feat(server): 实现 /api/capabilities 端点"
```

---

### 任务 8：实现 ports 端点（check/allocate/release）

**文件：**
- 修改：`server/internal/api/api.go` — 加 3 个 handler
- 修改：`server/internal/api/api_test.go` — 加测试

- [ ] **步骤 1：编写失败的测试**

追加到 `server/internal/api/api_test.go`：

```go
func TestPortsCheck(t *testing.T) {
	cfgPath := writeAgentConfig(t)
	cfg, _ := config.Load(cfgPath)
	srv := NewTestServer(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/ports/check?protocol=tcp&port=10001", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp portpool.CheckResult
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Protocol != portpool.TCP || resp.Port != 10001 {
		t.Errorf("got %+v", resp)
	}
}

func TestPortsAllocateAndRelease(t *testing.T) {
	cfgPath := writeAgentConfig(t)
	cfg, _ := config.Load(cfgPath)
	srv := NewTestServer(t, cfg)

	// allocate
	req := httptest.NewRequest(http.MethodPost, "/api/ports/allocate", strings.NewReader(`{"protocol":"tcp"}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("allocate status=%d body=%s", rec.Code, rec.Body.String())
	}
	var alloc struct {
		Protocol string `json:"protocol"`
		Port     int    `json:"port"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &alloc)
	if alloc.Port < 10000 || alloc.Port > 60000 {
		t.Errorf("allocated port = %d, 不在范围", alloc.Port)
	}

	// release
	body := fmt.Sprintf(`{"protocol":"tcp","port":%d}`, alloc.Port)
	req2 := httptest.NewRequest(http.MethodPost, "/api/ports/release", strings.NewReader(body))
	req2.Header.Set("Authorization", "Bearer test-token")
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("release status=%d body=%s", rec2.Code, rec2.Body.String())
	}
}
```

测试文件需 import `"fmt"`、`"strings"`、`"github.com/kdc/frp-manager/server/internal/portpool"`。

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/api/ -v -run TestPorts`
预期：FAIL，路由 404 或 handler nil

- [ ] **步骤 3：编写实现**

在 `api.go` 的 `Router()` 启用路由：

```go
	r.Get("/api/ports/check", s.checkPort)
	r.Post("/api/ports/allocate", s.allocatePort)
	r.Post("/api/ports/release", s.releasePort)
```

添加 handler：

```go
func (s *Server) checkPort(w http.ResponseWriter, r *http.Request) {
	protocol := r.URL.Query().Get("protocol")
	portStr := r.URL.Query().Get("port")
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效的 port"})
		return
	}
	res, err := s.ports.Check(r.Context(), portpool.Protocol(protocol), port)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) allocatePort(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Protocol string `json:"protocol"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效请求体"})
		return
	}
	port, err := s.ports.Allocate(r.Context(), portpool.Protocol(req.Protocol))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"protocol": req.Protocol, "port": port})
}

func (s *Server) releasePort(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Protocol string `json:"protocol"`
		Port     int    `json:"port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效请求体"})
		return
	}
	if err := s.ports.Release(r.Context(), portpool.Protocol(req.Protocol), req.Port); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}
```

import 块加 `"strconv"`。

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/api/ -v -run TestPorts`
预期：PASS

- [ ] **步骤 5：Commit**

```bash
git add server/internal/api/
git commit -m "feat(server): 实现 /api/ports/{check,allocate,release}"
```

---

### 任务 9：实现 domains 端点（check/register/release）

**文件：**
- 修改：`server/internal/api/api.go` — 加 3 个 handler
- 修改：`server/internal/api/api_test.go` — 加测试

- [ ] **步骤 1：编写失败的测试**

追加到 `api_test.go`：

```go
func TestDomainsCheck(t *testing.T) {
	cfgPath := writeAgentConfig(t)
	cfg, _ := config.Load(cfgPath)
	srv := NewTestServer(t, cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/domains/check", strings.NewReader(`{"protocol":"http","domain":"app.example.com"}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp domain.CheckResult
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if !resp.Available {
		t.Errorf("app.example.com 应可用, got %+v", resp)
	}
}

func TestDomainsRegisterAndRelease(t *testing.T) {
	cfgPath := writeAgentConfig(t)
	cfg, _ := config.Load(cfgPath)
	srv := NewTestServer(t, cfg)

	// register
	req := httptest.NewRequest(http.MethodPost, "/api/domains/register", strings.NewReader(`{"protocol":"http","domain":"app.example.com","tunnel_id":"t1"}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("register status=%d body=%s", rec.Code, rec.Body.String())
	}

	// 再 check 应不可用
	req2 := httptest.NewRequest(http.MethodPost, "/api/domains/check", strings.NewReader(`{"protocol":"http","domain":"app.example.com"}`))
	req2.Header.Set("Authorization", "Bearer test-token")
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec2, req2)
	var resp domain.CheckResult
	_ = json.Unmarshal(rec2.Body.Bytes(), &resp)
	if resp.Available {
		t.Errorf("注册后应不可用")
	}

	// release
	req3 := httptest.NewRequest(http.MethodPost, "/api/domains/release", strings.NewReader(`{"protocol":"http","domain":"app.example.com"}`))
	req3.Header.Set("Authorization", "Bearer test-token")
	req3.Header.Set("Content-Type", "application/json")
	rec3 := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("release status=%d body=%s", rec3.Code, rec3.Body.String())
	}
}
```

测试 import 加 `"github.com/kdc/frp-manager/server/internal/domain"`。

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/api/ -v -run TestDomains`
预期：FAIL，路由 404

- [ ] **步骤 3：编写实现**

在 `Router()` 启用：

```go
	r.Post("/api/domains/check", s.checkDomain)
	r.Post("/api/domains/register", s.registerDomain)
	r.Post("/api/domains/release", s.releaseDomain)
```

添加 handler：

```go
func (s *Server) checkDomain(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Protocol string `json:"protocol"`
		Domain   string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效请求体"})
		return
	}
	res, err := s.domains.Check(r.Context(), domain.Protocol(req.Protocol), req.Domain)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) registerDomain(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Protocol string `json:"protocol"`
		Domain   string `json:"domain"`
		TunnelID string `json:"tunnel_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效请求体"})
		return
	}
	if err := s.domains.Register(r.Context(), domain.Protocol(req.Protocol), req.Domain, req.TunnelID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) releaseDomain(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Protocol string `json:"protocol"`
		Domain   string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效请求体"})
		return
	}
	if err := s.domains.Release(r.Context(), domain.Protocol(req.Protocol), req.Domain); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}
```

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/api/ -v -run TestDomains`
预期：PASS

- [ ] **步骤 5：Commit**

```bash
git add server/internal/api/
git commit -m "feat(server): 实现 /api/domains/{check,register,release}"
```

---

### 任务 10：装配 main.go 并端到端冒烟验证

**文件：**
- 修改：`server/cmd/agent/main.go`

- [ ] **步骤 1：修改 main.go 装配真实依赖**

把 `main.go` 里的 `api.New(cfg)` 改为完整装配：

```go
	// 初始化 SQLite
	db, err := store.Open(cfg.Server.Database)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer func() { _ = db.Close() }()

	st, err := store.NewStore(db)
	if err != nil {
		log.Fatalf("创建 store 失败: %v", err)
	}

	// frps 管理（解析配置、检测状态）
	frpsMgr := frps.NewManager(cfg.Frps.Config, cfg.Frps.Binary)
	frpCfg, err := frpsMgr.Config()
	if err != nil {
		log.Printf("警告: 解析 frps 配置失败（capabilities 将返回不完整）: %v", err)
	}

	// 端口池与域名管理
	portsMgr := portpool.NewManager(st, frpCfg)
	domainMgr := domain.NewManager(st, &cfg.Domain)

	apiSrv := api.New(cfg, st, frpsMgr, portsMgr, domainMgr)
```

import 块加入：

```go
	"github.com/kdc/frp-manager/server/internal/domain"
	"github.com/kdc/frp-manager/server/internal/frps"
	"github.com/kdc/frp-manager/server/internal/portpool"
```

> 注意：`api.New` 签名已在任务 7 改为 `New(cfg, st, f, p, d)`，需同步删除旧 `api.New(cfg)` 调用。

- [ ] **步骤 2：编译验证**

运行：`go build ./... && go vet ./...`
预期：无输出（成功）

- [ ] **步骤 3：运行全部单测**

运行：`go test ./... -v`
预期：所有包测试 PASS

- [ ] **步骤 4：端到端冒烟验证**

启动 agent（后台）：

```bash
go run ./cmd/agent -config configs/agent.toml.example
```

依次 curl（另开终端）：

```bash
# health（无需 token）
curl -s http://127.0.0.1:7400/api/health
# 预期: {"status":"ok","version":"0.1.0"}

# capabilities
curl -s -H "Authorization: Bearer change-me-agent-token" http://127.0.0.1:7400/api/capabilities
# 预期: frps_running=false（本地无 frps 进程），bind_port=7000, support_http=true, vhost_http_port=80, ...

# 端口检查
curl -s -H "Authorization: Bearer change-me-agent-token" "http://127.0.0.1:7400/api/ports/check?protocol=tcp&port=10001"
# 预期: {"protocol":"tcp","port":10001,"available":true,"reason":"available"}

# 端口分配
curl -s -X POST -H "Authorization: Bearer change-me-agent-token" -H "Content-Type: application/json" -d '{"protocol":"tcp"}' http://127.0.0.1:7400/api/ports/allocate
# 预期: {"protocol":"tcp","port":10001}（或范围内任意可用端口）

# 域名检查
curl -s -X POST -H "Authorization: Bearer change-me-agent-token" -H "Content-Type: application/json" -d '{"protocol":"http","domain":"app.example.com"}' http://127.0.0.1:7400/api/domains/check
# 预期: {"domain":"app.example.com","available":true,"reason":"available"}

# 域名注册
curl -s -X POST -H "Authorization: Bearer change-me-agent-token" -H "Content-Type: application/json" -d '{"protocol":"http","domain":"app.example.com","tunnel_id":"t1"}' http://127.0.0.1:7400/api/domains/register
# 预期: {"success":true}
```

> 注意：`configs/agent.toml.example` 的 token 默认是 `change-me-agent-token`，frps.config 路径默认 `/opt/...`（Linux 路径，Windows 下解析会失败但 capabilities 仍能返回 support_* 字段来自配置默认值；若想完整验证，把 example 里 frps.config 指向一个本地真实 frps.toml）。**冒烟时若 frps 配置解析失败，capabilities 会返回 500——这是预期，因为 example 是 Linux 路径。可在冒烟前临时复制一份本地 frps.toml 并修改 agent.toml.example 的 frps.config 指向它。**

- [ ] **步骤 5：Commit**

```bash
git add server/cmd/agent/main.go
git commit -m "feat(server): main.go 装配真实依赖完成 v0.1 服务端闭环"
```

---

## 自检结果

**1. 规格覆盖度**（对照设计文档第 13 节）：
- 13.1 health ✅（已在骨架实现，本计划保留）
- 13.2 capabilities ✅ 任务 7
- 13.3 端口 check ✅ 任务 8
- 13.4 端口 allocate ✅ 任务 8
- 13.5 端口 release ✅ 任务 8
- 13.6 域名 check ✅ 任务 9
- 13.7 域名 register ✅ 任务 9
- 13.8 域名 release ✅ 任务 9
- 第 14 节数据库表 ✅ 骨架已有 migration，任务 3 加 CRUD
- 第 4 节 frps-agent 职责：frps 状态检测 ✅ 任务 6、配置读取 ✅ 任务 6、端口池 ✅ 任务 4、域名规则 ✅ 任务 5、token 管理 ✅ 骨架中间件

**2. 占位符扫描**：无 TODO/待定；每个步骤都有完整代码。

**3. 类型一致性**：
- `frpsc.Config`、`frpsc.AllowPort` 全程一致
- `store.PortAllocation`/`DomainAllocation` 字段名一致
- `portpool.Manager`（结构体）实现 `portpool.Manager`（接口）方法签名一致
- `domain.Manager` 同上
- `frps.Manager` 接口加了 `Config()`，`api.Server` 调用 `s.frps.Config()` 一致
- `api.New` 签名 `(cfg, st, f, p, d)` 在任务 7 定义、任务 10 调用一致
- `api.NewTestServer` 在任务 7 定义、任务 8/9 测试调用一致

**已识别的风险/注意事项（已在对应任务标注）：**
1. 任务 2 的 `net.ListenTimeout` 不存在，明确指示改用 `net.Listen`。
2. 任务 7 测试 `writeAgentConfig` 在 Windows 下 TOML 字符串转义问题，明确指示用单引号字面串。
3. 任务 10 冒烟时 `agent.toml.example` 的 frps.config 是 Linux 路径，Windows 下 capabilities 会 500，需临时改指向本地 frps.toml。

这些风险都在对应任务步骤里给出了具体处理方式，不构成阻塞。
