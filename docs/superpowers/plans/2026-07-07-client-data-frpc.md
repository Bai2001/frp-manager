# 客户端数据层与 frpc 实现计划（v0.1 计划 2/3）

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 把 `client/` 骨架的占位实现替换为真实的数据层与 frpc 控制，覆盖：servers/tunnels 的 SQLite CRUD、调用 server-agent 全部 HTTP API 的 client、frpc.toml 生成（修正 toml 嵌套结构）、frpc 进程启停重启、`App` 方法接通真实依赖。完成后客户端可通过 Wails 绑定方法管理服务器与映射、生成配置、控制 frpc，但不涉及前端 UI 联动（计划 3 负责）。

**架构：** 复用 `client/` module（`github.com/kdc/frp-manager/client`）。新增 `client/internal/db/repo.go`（CRUD）、改造 `client/internal/agent`（补全 8 个端点）、`client/internal/frpc`（toml 嵌套结构修正 + 配置生成 + 进程管理）、`client/app.go`（注入依赖、实现方法）。所有外部依赖通过构造函数注入，`App` 持有 `*db.Repo`、按需创建 `agent.Client`、持有 `*frpc.Manager`。

**技术栈：** Go 1.26、`modernc.org/sqlite`（已引入）、`github.com/google/uuid`（需显式 require）、`net/http` 标准库（agent client）、`os/exec`（frpc 进程）、`github.com/pelletier/go-toml/v2`（frpc.toml 生成，需显式 require）。依赖计划 1 的 server-agent API 契约（端点路径与请求/响应体）。

**约定：**
- 时间戳统一 `time.Now().UTC().Format(time.RFC3339)`，存 SQLite DATETIME 文本。
- ID 用 `uuid.NewString()`。
- `is_default`/`enabled` 在 Go 层用 `bool`，DB 层用 INTEGER 0/1，repo 做转换。
- frpc 进程每个 server 一个，`Manager` 内部维护 `map[serverID]*exec.Cmd` + 配置文件路径。
- 本计划所有任务用 `go test` 验证，不启动 Wails。

---

## 文件结构

**新建：**
- `client/internal/db/repo.go` — `Repo` 封装 servers/tunnels 的 CRUD
- `client/internal/db/repo_test.go`
- `client/internal/agent/ports.go` — 端口相关 DTO 与方法
- `client/internal/agent/domains.go` — 域名相关 DTO 与方法
- `client/internal/frpc/config.go` — 修正后的 toml 结构 + Generate 实现
- `client/internal/frpc/config_test.go`
- `client/internal/frpc/manager.go` — 进程管理真实实现
- `client/internal/frpc/manager_test.go`

**修改：**
- `client/internal/agent/agent.go` — 补全 Health/Capabilities/CheckPort/AllocatePort/ReleasePort/CheckDomain/RegisterDomain/ReleaseDomain
- `client/internal/frpc/frpc.go` — 删除旧 `Config`/`Proxy`/`noopManager`，类型移到 `config.go`，接口保留
- `client/app.go` — 注入 `*db.Repo`、`*frpc.Manager`，实现全部方法，新增 AddServer/UpdateServer/DeleteServer/AddTunnel/UpdateTunnel/DeleteTunnel/CheckServerCapabilities
- `client/types.go` — 新增 AddServerInput/AddTunnelInput 输入类型
- `client/main.go` — startup 时初始化 db/frpc 并注入 App
- `client/go.mod` — require `github.com/google/uuid`、`github.com/pelletier/go-toml/v2`

**测试策略：** db repo 用临时 SQLite 文件单测；agent client 用 `httptest` 起假 server 验证请求路径与响应解析；frpc Generate 用字符串断言；frpc 进程管理用真实 `frpc -v` 或 `cmd /c echo` 替身验证启停（Windows 下用 `ping -n 2 127.0.0.1` 作长进程替身）。

---

### 任务 1：db CRUD（repo.go）

**文件：**
- 创建：`client/internal/db/repo.go`、`client/internal/db/repo_test.go`

- [ ] **步骤 1：编写失败的测试**

`client/internal/db/repo_test.go`：

```go
package db

import (
	"path/filepath"
	"testing"
	"time"
)

func newRepo(t *testing.T) *Repo {
	t.Helper()
	db := Open(filepath.Join(t.TempDir(), "test.db"))
	r, err := NewRepo(db)
	if err != nil {
		t.Fatalf("NewRepo: %v", err)
	}
	return r
}

func TestServers_CRUD(t *testing.T) {
	r := newRepo(t)
	now := time.Now().UTC()
	s := Server{
		ID: "s1", Name: "prod", Host: "1.2.3.4", FrpsPort: 7000,
		FrpToken: "tok", AgentURL: "http://1.2.3.4:7400", AgentToken: "atok",
		IsDefault: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := r.InsertServer(s); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, err := r.GetServer("s1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "prod" || !got.IsDefault {
		t.Errorf("got %+v", got)
	}
	all, err := r.ListServers()
	if err != nil || len(all) != 1 {
		t.Errorf("ListServers len = %d, err %v", len(all), err)
	}
	s.Name = "prod2"
	if err := r.UpdateServer(s); err != nil {
		t.Fatal(err)
	}
	got, _ = r.GetServer("s1")
	if got.Name != "prod2" {
		t.Errorf("name = %q", got.Name)
	}
	if err := r.DeleteServer("s1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetServer("s1"); err == nil {
		t.Error("删除后应查不到")
	}
}

func TestTunnels_CRUD(t *testing.T) {
	r := newRepo(t)
	now := time.Now().UTC()
	_ = r.InsertServer(Server{
		ID: "s1", Name: "prod", Host: "1.2.3.4", FrpsPort: 7000,
		FrpToken: "t", AgentURL: "http://x", AgentToken: "a",
		CreatedAt: now, UpdatedAt: now,
	})
	tu := Tunnel{
		ID: "t1", ServerID: "s1", Name: "rdp", Protocol: "tcp",
		LocalIP: "127.0.0.1", LocalPort: 3389, RemotePort: 20389,
		Enabled: true, Status: "stopped", CreatedAt: now, UpdatedAt: now,
	}
	if err := r.InsertTunnel(tu); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, err := r.GetTunnel("t1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Protocol != "tcp" || got.RemotePort != 20389 || !got.Enabled {
		t.Errorf("got %+v", got)
	}
	list, err := r.ListTunnelsByServer("s1")
	if err != nil || len(list) != 1 {
		t.Errorf("ListTunnelsByServer len = %d", len(list))
	}
	tu.Status = "running"
	if err := r.UpdateTunnel(tu); err != nil {
		t.Fatal(err)
	}
	got, _ = r.GetTunnel("t1")
	if got.Status != "running" {
		t.Errorf("status = %q", got.Status)
	}
	if err := r.DeleteTunnel("t1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetTunnel("t1"); err == nil {
		t.Error("删除后应查不到")
	}
}

func TestTunnels_DeleteServerCascade(t *testing.T) {
	r := newRepo(t)
	now := time.Now().UTC()
	_ = r.InsertServer(Server{ID: "s1", Name: "p", Host: "h", FrpsPort: 7000, FrpToken: "t", AgentURL: "u", AgentToken: "a", CreatedAt: now, UpdatedAt: now})
	_ = r.InsertTunnel(Tunnel{ID: "t1", ServerID: "s1", Name: "n", Protocol: "tcp", LocalIP: "127.0.0.1", LocalPort: 22, Enabled: true, Status: "stopped", CreatedAt: now, UpdatedAt: now})
	if err := r.DeleteServer("s1"); err != nil {
		t.Fatal(err)
	}
	list, _ := r.ListTunnelsByServer("s1")
	if len(list) != 0 {
		t.Errorf("删除 server 后 tunnels 应级联清空, len=%d", len(list))
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/db/ -v`
预期：FAIL，`undefined: Repo`、`undefined: Server` 等

- [ ] **步骤 3：编写实现**

`client/internal/db/repo.go`：

```go
package db

import (
	"database/sql"
	"fmt"
	"time"
)

// Server 对应 servers 表一行。
type Server struct {
	ID         string
	Name       string
	Host       string
	FrpsPort   int
	FrpToken   string
	AgentURL   string
	AgentToken string
	IsDefault  bool
	Remark     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Tunnel 对应 tunnels 表一行。
type Tunnel struct {
	ID           string
	ServerID     string
	Name         string
	Protocol     string
	LocalIP      string
	LocalPort    int
	RemotePort   int
	CustomDomain string
	Subdomain    string
	Enabled      bool
	Status       string
	LastError    string
	Remark       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Repo 封装 servers/tunnels 的数据库操作。
type Repo struct {
	db *sql.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *sql.DB) (*Repo, error) {
	return &Repo{db: db}, nil
}

// InsertServer 插入一条服务器。
func (r *Repo) InsertServer(s Server) error {
	_, err := r.db.Exec(
		`INSERT INTO servers (id,name,host,frps_port,frp_token,agent_url,agent_token,is_default,remark,created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		s.ID, s.Name, s.Host, s.FrpsPort, s.FrpToken, s.AgentURL, s.AgentToken,
		boolToInt(s.IsDefault), s.Remark, s.CreatedAt.Format(time.RFC3339), s.UpdatedAt.Format(time.RFC3339))
	return err
}

// GetServer 查询单个服务器。
func (r *Repo) GetServer(id string) (*Server, error) {
	row := r.db.QueryRow(
		`SELECT id,name,host,frps_port,frp_token,agent_url,agent_token,is_default,remark,created_at,updated_at
		 FROM servers WHERE id=?`, id)
	var s Server
	var isDefault int
	var created, updated string
	if err := row.Scan(&s.ID, &s.Name, &s.Host, &s.FrpsPort, &s.FrpToken, &s.AgentURL, &s.AgentToken, &isDefault, &s.Remark, &created, &updated); err != nil {
		return nil, err
	}
	s.IsDefault = isDefault == 1
	s.CreatedAt, _ = time.Parse(time.RFC3339, created)
	s.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &s, nil
}

// ListServers 返回全部服务器。
func (r *Repo) ListServers() ([]Server, error) {
	rows, err := r.db.Query(
		`SELECT id,name,host,frps_port,frp_token,agent_url,agent_token,is_default,remark,created_at,updated_at
		 FROM servers ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Server
	for rows.Next() {
		var s Server
		var isDefault int
		var created, updated string
		if err := rows.Scan(&s.ID, &s.Name, &s.Host, &s.FrpsPort, &s.FrpToken, &s.AgentURL, &s.AgentToken, &isDefault, &s.Remark, &created, &updated); err != nil {
			return nil, err
		}
		s.IsDefault = isDefault == 1
		s.CreatedAt, _ = time.Parse(time.RFC3339, created)
		s.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		out = append(out, s)
	}
	return out, rows.Err()
}

// UpdateServer 更新服务器（按 ID）。
func (r *Repo) UpdateServer(s Server) error {
	_, err := r.db.Exec(
		`UPDATE servers SET name=?,host=?,frps_port=?,frp_token=?,agent_url=?,agent_token=?,is_default=?,remark=?,updated_at=? WHERE id=?`,
		s.Name, s.Host, s.FrpsPort, s.FrpToken, s.AgentURL, s.AgentToken,
		boolToInt(s.IsDefault), s.Remark, s.UpdatedAt.Format(time.RFC3339), s.ID)
	return err
}

// DeleteServer 删除服务器，并级联删除其下 tunnels。
func (r *Repo) DeleteServer(id string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM tunnels WHERE server_id=?`, id); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`DELETE FROM servers WHERE id=?`, id); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// InsertTunnel 插入一条映射。
func (r *Repo) InsertTunnel(t Tunnel) error {
	_, err := r.db.Exec(
		`INSERT INTO tunnels (id,server_id,name,protocol,local_ip,local_port,remote_port,custom_domain,subdomain,enabled,status,last_error,remark,created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.ServerID, t.Name, t.Protocol, t.LocalIP, t.LocalPort, t.RemotePort,
		t.CustomDomain, t.Subdomain, boolToInt(t.Enabled), t.Status, t.LastError, t.Remark,
		t.CreatedAt.Format(time.RFC3339), t.UpdatedAt.Format(time.RFC3339))
	return err
}

// GetTunnel 查询单个映射。
func (r *Repo) GetTunnel(id string) (*Tunnel, error) {
	row := r.db.QueryRow(
		`SELECT id,server_id,name,protocol,local_ip,local_port,remote_port,custom_domain,subdomain,enabled,status,last_error,remark,created_at,updated_at
		 FROM tunnels WHERE id=?`, id)
	var t Tunnel
	var enabled int
	var created, updated string
	if err := row.Scan(&t.ID, &t.ServerID, &t.Name, &t.Protocol, &t.LocalIP, &t.LocalPort, &t.RemotePort, &t.CustomDomain, &t.Subdomain, &enabled, &t.Status, &t.LastError, &t.Remark, &created, &updated); err != nil {
		return nil, err
	}
	t.Enabled = enabled == 1
	t.CreatedAt, _ = time.Parse(time.RFC3339, created)
	t.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &t, nil
}

// ListTunnelsByServer 返回指定服务器的映射。
func (r *Repo) ListTunnelsByServer(serverID string) ([]Tunnel, error) {
	rows, err := r.db.Query(
		`SELECT id,server_id,name,protocol,local_ip,local_port,remote_port,custom_domain,subdomain,enabled,status,last_error,remark,created_at,updated_at
		 FROM tunnels WHERE server_id=? ORDER BY created_at`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tunnel
	for rows.Next() {
		var t Tunnel
		var enabled int
		var created, updated string
		if err := rows.Scan(&t.ID, &t.ServerID, &t.Name, &t.Protocol, &t.LocalIP, &t.LocalPort, &t.RemotePort, &t.CustomDomain, &t.Subdomain, &enabled, &t.Status, &t.LastError, &t.Remark, &created, &updated); err != nil {
			return nil, err
		}
		t.Enabled = enabled == 1
		t.CreatedAt, _ = time.Parse(time.RFC3339, created)
		t.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		out = append(out, t)
	}
	return out, rows.Err()
}

// UpdateTunnel 更新映射。
func (r *Repo) UpdateTunnel(t Tunnel) error {
	_, err := r.db.Exec(
		`UPDATE tunnels SET name=?,protocol=?,local_ip=?,local_port=?,remote_port=?,custom_domain=?,subdomain=?,enabled=?,status=?,last_error=?,remark=?,updated_at=? WHERE id=?`,
		t.Name, t.Protocol, t.LocalIP, t.LocalPort, t.RemotePort, t.CustomDomain, t.Subdomain,
		boolToInt(t.Enabled), t.Status, t.LastError, t.Remark, t.UpdatedAt.Format(time.RFC3339), t.ID)
	return err
}

// UpdateTunnelStatus 仅更新状态与错误信息（frpc 进程管理常用）。
func (r *Repo) UpdateTunnelStatus(id, status, lastError string, at time.Time) error {
	_, err := r.db.Exec(`UPDATE tunnels SET status=?, last_error=?, updated_at=? WHERE id=?`,
		status, lastError, at.Format(time.RFC3339), id)
	return err
}

// DeleteTunnel 删除映射。
func (r *Repo) DeleteTunnel(id string) error {
	_, err := r.db.Exec(`DELETE FROM tunnels WHERE id=?`, id)
	return err
}

// ListAllTunnels 返回全部映射（生成配置时按 server 分组用）。
func (r *Repo) ListAllTunnels() ([]Tunnel, error) {
	rows, err := r.db.Query(
		`SELECT id,server_id,name,protocol,local_ip,local_port,remote_port,custom_domain,subdomain,enabled,status,last_error,remark,created_at,updated_at
		 FROM tunnels ORDER BY server_id, created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tunnel
	for rows.Next() {
		var t Tunnel
		var enabled int
		var created, updated string
		if err := rows.Scan(&t.ID, &t.ServerID, &t.Name, &t.Protocol, &t.LocalIP, &t.LocalPort, &t.RemotePort, &t.CustomDomain, &t.Subdomain, &enabled, &t.Status, &t.LastError, &t.Remark, &created, &updated); err != nil {
			return nil, err
		}
		t.Enabled = enabled == 1
		t.CreatedAt, _ = time.Parse(time.RFC3339, created)
		t.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		out = append(out, t)
	}
	return out, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

var _ = fmt.Sprintf
```

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/db/ -v`
预期：PASS，3 个测试全过

- [ ] **步骤 5：Commit**

```bash
git add client/internal/db/
git commit -m "feat(client): db 新增 servers/tunnels CRUD repo"
```

---

### 任务 2：agent HTTP client 补全全部端点

**文件：**
- 修改：`client/internal/agent/agent.go`
- 创建：`client/internal/agent/ports.go`、`client/internal/agent/domains.go`、`client/internal/agent/agent_test.go`

- [ ] **步骤 1：编写失败的测试**

`client/internal/agent/agent_test.go`：

```go
package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return New(srv.URL, "test-token")
}

func TestHealth(t *testing.T) {
	called := false
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Path != "/api/health" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": "0.1.0"})
	})
	if err := c.Health(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("未调用 health")
	}
}

func TestCapabilities(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/capabilities" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("auth header = %q", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(Capabilities{BindPort: 7000, SupportTCP: true})
	})
	caps, err := c.Capabilities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if caps.BindPort != 7000 || !caps.SupportTCP {
		t.Errorf("got %+v", caps)
	}
}

func TestCheckPort(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("protocol") != "tcp" || r.URL.Query().Get("port") != "10001" {
			t.Errorf("query = %v", r.URL.Query())
		}
		_ = json.NewEncoder(w).Encode(PortCheckResult{Protocol: "tcp", Port: 10001, Available: true, Reason: "available"})
	})
	res, err := c.CheckPort(context.Background(), "tcp", 10001)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Available {
		t.Errorf("got %+v", res)
	}
}

func TestAllocatePort(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Protocol string `json:"protocol"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Protocol != "tcp" {
			t.Errorf("protocol = %q", req.Protocol)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"protocol": "tcp", "port": 10001})
	})
	port, err := c.AllocatePort(context.Background(), "tcp")
	if err != nil {
		t.Fatal(err)
	}
	if port != 10001 {
		t.Errorf("port = %d", port)
	}
}

func TestReleasePort(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	})
	if err := c.ReleasePort(context.Background(), "tcp", 10001); err != nil {
		t.Fatal(err)
	}
}

func TestCheckDomain(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(DomainCheckResult{Domain: "app.example.com", Available: true, Reason: "available"})
	})
	res, err := c.CheckDomain(context.Background(), "http", "app.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Available {
		t.Errorf("got %+v", res)
	}
}

func TestRegisterDomain(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	})
	if err := c.RegisterDomain(context.Background(), "http", "app.example.com", "t1"); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseDomain(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	})
	if err := c.ReleaseDomain(context.Background(), "http", "app.example.com"); err != nil {
		t.Fatal(err)
	}
}

func TestErrorOn500(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "boom"})
	})
	if err := c.Health(context.Background()); err == nil {
		t.Error("500 应返回 error")
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/agent/ -v`
预期：FAIL，`undefined: PortCheckResult`、`undefined: c.CheckPort` 等

- [ ] **步骤 3：编写实现**

`client/internal/agent/agent.go`（整体替换）：

```go
// Package agent 是调用 server-agent HTTP API 的客户端。
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Capabilities 对应 GET /api/capabilities 响应。
type Capabilities struct {
	FrpsRunning        bool             `json:"frps_running"`
	FrpsVersion        string           `json:"frps_version"`
	BindPort           int              `json:"bind_port"`
	AllowPorts         []AllowPortRange `json:"allow_ports"`
	SupportTCP         bool             `json:"support_tcp"`
	SupportUDP         bool             `json:"support_udp"`
	SupportHTTP        bool             `json:"support_http"`
	SupportHTTPS       bool             `json:"support_https"`
	VhostHTTPPort      int              `json:"vhost_http_port"`
	VhostHTTPSPort     int              `json:"vhost_https_port"`
	SubdomainHost      string           `json:"subdomain_host"`
	AllowedRootDomains []string         `json:"allowed_root_domains"`
}

// AllowPortRange 是 capabilities 响应里的端口范围项。
type AllowPortRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// Client 调用 server-agent API。
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// New 创建 agent 客户端。
func New(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Health 检查 agent 是否可达（GET /api/health，无需 token）。
func (c *Client) Health(ctx context.Context) error {
	_, err := c.do(ctx, http.MethodGet, "/api/health", nil, nil, false)
	return err
}

// Capabilities 查询服务端能力。
func (c *Client) Capabilities(ctx context.Context) (*Capabilities, error) {
	var caps Capabilities
	if err := c.getJSON(ctx, "/api/capabilities", &caps); err != nil {
		return nil, err
	}
	return &caps, nil
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	return c.decode(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) decode(ctx context.Context, method, path string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(buf)
	}
	resp, err := c.do(ctx, method, path, bodyReader, body != nil, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("解析响应: %w", err)
		}
	}
	return nil
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader, hasBody, withAuth bool) (*http.Response, error) {
	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, err
	}
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	if withAuth {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 %s: %w", path, err)
	}
	if resp.StatusCode >= 400 {
		var errBody struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		resp.Body.Close()
		msg := errBody.Error
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("agent %d: %s", resp.StatusCode, msg)
	}
	return resp, nil
}

func (c *Client) buildURL(path string, params map[string]string) string {
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	return path + "?" + q.Encode()
}

var _ = strconv.Itoa
```

`client/internal/agent/ports.go`：

```go
package agent

import "context"

// PortCheckResult 对应 GET /api/ports/check 响应。
type PortCheckResult struct {
	Protocol  string `json:"protocol"`
	Port      int    `json:"port"`
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
}

// PortAllocateResult 对应 POST /api/ports/allocate 响应。
type PortAllocateResult struct {
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
}

// CheckPort 检查端口可用性。
func (c *Client) CheckPort(ctx context.Context, protocol string, port int) (*PortCheckResult, error) {
	var res PortCheckResult
	path := c.buildURL("/api/ports/check", map[string]string{"protocol": protocol, "port": strconv.Itoa(port)})
	if err := c.getJSON(ctx, path, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// AllocatePort 自动分配一个端口。
func (c *Client) AllocatePort(ctx context.Context, protocol string) (int, error) {
	var res PortAllocateResult
	if err := c.decode(ctx, http.MethodPost, "/api/ports/allocate", map[string]string{"protocol": protocol}, &res); err != nil {
		return 0, err
	}
	return res.Port, nil
}

// ReleasePort 释放端口。
func (c *Client) ReleasePort(ctx context.Context, protocol string, port int) error {
	return c.decode(ctx, http.MethodPost, "/api/ports/release",
		map[string]any{"protocol": protocol, "port": port}, nil)
}
```

`ports.go` 顶部 import 块加 `"net/http"`、`"strconv"`。

`client/internal/agent/domains.go`：

```go
package agent

import "context"

// DomainCheckResult 对应 POST /api/domains/check 响应。
type DomainCheckResult struct {
	Domain    string `json:"domain"`
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
}

// CheckDomain 校验域名可用性。
func (c *Client) CheckDomain(ctx context.Context, protocol, domain string) (*DomainCheckResult, error) {
	var res DomainCheckResult
	if err := c.decode(ctx, http.MethodPost, "/api/domains/check",
		map[string]string{"protocol": protocol, "domain": domain}, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// RegisterDomain 注册域名占用。
func (c *Client) RegisterDomain(ctx context.Context, protocol, domain, tunnelID string) error {
	return c.decode(ctx, http.MethodPost, "/api/domains/register",
		map[string]string{"protocol": protocol, "domain": domain, "tunnel_id": tunnelID}, nil)
}

// ReleaseDomain 释放域名。
func (c *Client) ReleaseDomain(ctx context.Context, protocol, domain string) error {
	return c.decode(ctx, http.MethodPost, "/api/domains/release",
		map[string]string{"protocol": protocol, "domain": domain}, nil)
}
```

`domains.go` 顶部 import 块加 `"net/http"`。

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/agent/ -v`
预期：PASS，9 个测试全过

- [ ] **步骤 5：Commit**

```bash
git add client/internal/agent/
git commit -m "feat(client): agent client 补全 8 个端点 + httptest 单测"
```

---

### 任务 3：frpc.toml 配置生成（修正 toml 嵌套结构）

**文件：**
- 创建：`client/internal/frpc/config.go`、`client/internal/frpc/config_test.go`
- 修改：`client/internal/frpc/frpc.go` — 删除旧 Config/Proxy/noopManager，保留接口

- [ ] **步骤 1：编写失败的测试**

`client/internal/frpc/config_test.go`：

```go
package frpc

import (
	"strings"
	"testing"
)

func TestGenerate_TCPPort(t *testing.T) {
	cfg := &Config{
		ServerAddr: "1.2.3.4",
		ServerPort: 7000,
		Auth: Auth{Token: "tok"},
		Proxies: []Proxy{{
			Name: "rdp", Type: "tcp",
			LocalIP: "127.0.0.1", LocalPort: 3389, RemotePort: 20389,
		}},
	}
	out, err := Generate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `serverAddr = "1.2.3.4"`) {
		t.Errorf("缺少 serverAddr: %s", out)
	}
	if !strings.Contains(out, `serverPort = 7000`) {
		t.Errorf("缺少 serverPort: %s", out)
	}
	if !strings.Contains(out, `[auth]`) {
		t.Errorf("缺少 [auth] 表: %s", out)
	}
	if !strings.Contains(out, `token = "tok"`) {
		t.Errorf("缺少 auth.token: %s", out)
	}
	if !strings.Contains(out, `[[proxies]]`) {
		t.Errorf("缺少 [[proxies]]: %s", out)
	}
	if !strings.Contains(out, `remotePort = 20389`) {
		t.Errorf("缺少 remotePort: %s", out)
	}
}

func TestGenerate_HTTPDomain(t *testing.T) {
	cfg := &Config{
		ServerAddr: "1.2.3.4", ServerPort: 7000,
		Proxies: []Proxy{{
			Name: "web", Type: "http",
			LocalIP: "127.0.0.1", LocalPort: 3000,
			CustomDomains: []string{"app.example.com"},
		}},
	}
	out, err := Generate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `type = "http"`) {
		t.Errorf("缺少 type http: %s", out)
	}
	if !strings.Contains(out, `customDomains = ["app.example.com"]`) {
		t.Errorf("缺少 customDomains: %s", out)
	}
}

func TestGenerate_HTTPSSubdomain(t *testing.T) {
	cfg := &Config{
		ServerAddr: "1.2.3.4", ServerPort: 7000,
		Proxies: []Proxy{{
			Name: "demo", Type: "https",
			LocalIP: "127.0.0.1", LocalPort: 8443,
			Subdomain: "demo",
		}},
	}
	out, err := Generate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `subdomain = "demo"`) {
		t.Errorf("缺少 subdomain: %s", out)
	}
}

func TestGenerate_UDP(t *testing.T) {
	cfg := &Config{
		ServerAddr: "1.2.3.4", ServerPort: 7000,
		Proxies: []Proxy{{
			Name: "wg", Type: "udp",
			LocalIP: "127.0.0.1", LocalPort: 51820, RemotePort: 25180,
		}},
	}
	out, err := Generate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `type = "udp"`) || !strings.Contains(out, `remotePort = 25180`) {
		t.Errorf("缺少 udp 配置: %s", out)
	}
}

func TestGenerate_EmptyProxies(t *testing.T) {
	cfg := &Config{ServerAddr: "1.2.3.4", ServerPort: 7000}
	out, err := Generate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "[[proxies]]") {
		t.Errorf("无 proxy 不应输出 [[proxies]]: %s", out)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/frpc/ -v`
预期：FAIL，`undefined: Config`、`undefined: Auth`、`undefined: Generate`

- [ ] **步骤 3：编写实现**

`client/internal/frpc/config.go`：

```go
package frpc

import (
	"bytes"
	"fmt"

	"github.com/pelletier/go-toml/v2"
)

// Config 是 frpc.toml 的完整结构。
// Auth 是嵌套 [auth] 表，避免点路径 tag 导致的 toml 写入问题。
type Config struct {
	ServerAddr string  `toml:"serverAddr"`
	ServerPort int     `toml:"serverPort"`
	Auth       Auth    `toml:"auth"`
	Proxies    []Proxy `toml:"proxies"`
}

// Auth 对应 [auth] 表。
type Auth struct {
	Method string `toml:"method,omitempty"`
	Token  string `toml:"token,omitempty"`
}

// Proxy 对应 [[proxies]] 段。
type Proxy struct {
	Name          string   `toml:"name"`
	Type          string   `toml:"type"`
	LocalIP       string   `toml:"localIP"`
	LocalPort     int      `toml:"localPort"`
	RemotePort    int      `toml:"remotePort,omitempty"`
	CustomDomains []string `toml:"customDomains,omitempty"`
	Subdomain     string   `toml:"subdomain,omitempty"`
}

// Generate 把 Config 序列化为 frpc.toml 文本。
// 空的 Proxies 不会输出 [[proxies]] 段。
func Generate(cfg *Config) (string, error) {
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	enc.SetIndentSymbol("  ")
	if err := enc.Encode(cfg); err != nil {
		return "", fmt.Errorf("序列化 frpc.toml: %w", err)
	}
	return buf.String(), nil
}
```

- [ ] **步骤 4：更新 frpc.go 删除旧类型与 noopManager**

`client/internal/frpc/frpc.go` 改为：

```go
// Package frpc 负责 frpc.toml 配置生成与 frpc 进程启停重启。
package frpc

import "context"

// Manager 抽象 frpc 的配置生成与进程控制（由 manager.go 实现）。
type Manager interface {
	Generate(cfg *Config) (string, error)
	Start(ctx context.Context, serverID string, cfgText string) error
	Stop(ctx context.Context, serverID string) error
	Restart(ctx context.Context, serverID string, cfgText string) error
}
```

> 注意：`Manager` 接口签名调整为带 `serverID` 与 `cfgText`，与新 manager.go 一致。原骨架 `Generate(cfg)` 签名保留，进程方法新增参数。

- [ ] **步骤 5：运行测试验证通过**

运行：`go test ./internal/frpc/ -v`
预期：PASS，5 个测试全过

- [ ] **步骤 6：Commit**

```bash
git add client/internal/frpc/config.go client/internal/frpc/config_test.go client/internal/frpc/frpc.go
git commit -m "feat(client): frpc 配置生成修正 toml 嵌套结构 + 单测"
```

---

### 任务 4：frpc 进程管理（manager.go）

**文件：**
- 创建：`client/internal/frpc/manager.go`、`client/internal/frpc/manager_test.go`

- [ ] **步骤 1：编写失败的测试**

`client/internal/frpc/manager_test.go`：

```go
package frpc

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// 替身进程：Windows 用 ping，Unix 用 sleep，足够长以验证 Stop 能终止。
func fakeBinary(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		return "ping"
	}
	return "sleep"
}

func fakeArgs() []string {
	if runtime.GOOS == "windows" {
		return []string{"-n", "30", "127.0.0.1"}
	}
	return []string{"30"}
}

func TestStartAndStop(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)
	m.SetBinary(fakeBinary(t), fakeArgs()...)

	if err := m.Start(context.Background(), "s1", "dummy config"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	// 给进程一点启动时间
	time.Sleep(200 * time.Millisecond)
	if !m.IsRunning("s1") {
		t.Errorf("启动后应 running")
	}
	if err := m.Stop(context.Background(), "s1"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	if m.IsRunning("s1") {
		t.Errorf("停止后不应 running")
	}
}

func TestStopNotRunning(t *testing.T) {
	m := NewManager(t.TempDir())
	if err := m.Stop(context.Background(), "s1"); err == nil {
		t.Error("停止未运行的进程应报错")
	}
}

func TestRestart(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)
	m.SetBinary(fakeBinary(t), fakeArgs()...)
	_ = m.Start(context.Background(), "s1", "v1")
	time.Sleep(200 * time.Millisecond)
	if err := m.Restart(context.Background(), "s1", "v2"); err != nil {
		t.Fatalf("Restart: %v", err)
	}
	if !m.IsRunning("s1") {
		t.Errorf("重启后应 running")
	}
	_ = m.Stop(context.Background(), "s1")
}

func TestConfigFileWritten(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)
	m.SetBinary(fakeBinary(t), fakeArgs()...)
	if err := m.Start(context.Background(), "s1", "serverAddr = \"1.2.3.4\""); err != nil {
		t.Fatal(err)
	}
	_ = m.Stop(context.Background(), "s1")
	// 配置文件应写入到 dir 下
	cfgPath := filepath.Join(dir, "s1.toml")
	if _, err := readFile(cfgPath); err != nil {
		t.Errorf("配置文件未写入: %v", err)
	}
}

func readFile(p string) (string, error) {
	b, err := osReadFile(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/frpc/ -v`
预期：FAIL，`undefined: NewManager`、`undefined: m.SetBinary`、`undefined: osReadFile`

- [ ] **步骤 3：编写实现**

`client/internal/frpc/manager.go`：

```go
package frpc

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// Manager 管理 frpc 进程，每个 serverID 对应一个进程 + 配置文件。
type Manager struct {
	mu        sync.Mutex
	procs     map[string]*exec.Cmd
	configDir string
	binary    string
	args      []string
}

// NewManager 创建进程管理器，configDir 用于存放每个 server 的 frpc.toml。
func NewManager(configDir string) *Manager {
	return &Manager{
		procs:     map[string]*exec.Cmd{},
		configDir: configDir,
		binary:    "frpc",
		args:      []string{"-c"},
	}
}

// SetBinary 覆盖默认 frpc 二进制与参数（测试用替身）。
func (m *Manager) SetBinary(name string, args ...string) {
	m.binary = name
	m.args = args
}

// Generate 委托给包级 Generate 函数。
func (m *Manager) Generate(cfg *Config) (string, error) {
	return Generate(cfg)
}

// Start 为指定 server 写入配置并启动 frpc。
func (m *Manager) Start(ctx context.Context, serverID, cfgText string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cmd, ok := m.procs[serverID]; ok && cmd.ProcessState == nil {
		return fmt.Errorf("server %s 的 frpc 已在运行", serverID)
	}
	if err := os.MkdirAll(m.configDir, 0o755); err != nil {
		return fmt.Errorf("创建配置目录: %w", err)
	}
	cfgPath := filepath.Join(m.configDir, serverID+".toml")
	if err := os.WriteFile(cfgPath, []byte(cfgText), 0o644); err != nil {
		return fmt.Errorf("写入 frpc 配置: %w", err)
	}
	args := append([]string{}, m.args...)
	args = append(args, cfgPath)
	cmd := exec.CommandContext(ctx, m.binary, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 frpc: %w", err)
	}
	m.procs[serverID] = cmd
	go func() { _ = cmd.Wait() }()
	return nil
}

// Stop 终止指定 server 的 frpc 进程。
func (m *Manager) Stop(_ context.Context, serverID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cmd, ok := m.procs[serverID]
	if !ok || cmd.Process == nil {
		return fmt.Errorf("server %s 未运行", serverID)
	}
	if err := cmd.Process.Kill(); err != nil {
		return fmt.Errorf("终止 frpc: %w", err)
	}
	delete(m.procs, serverID)
	return nil
}

// Restart 重启指定 server 的 frpc。
func (m *Manager) Restart(ctx context.Context, serverID, cfgText string) error {
	m.mu.Lock()
	if cmd, ok := m.procs[serverID]; ok && cmd.Process != nil {
		_ = cmd.Process.Kill()
		delete(m.procs, serverID)
	}
	m.mu.Unlock()
	return m.Start(ctx, serverID, cfgText)
}

// IsRunning 返回指定 server 的 frpc 是否在运行。
func (m *Manager) IsRunning(serverID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	cmd, ok := m.procs[serverID]
	if !ok {
		return false
	}
	return cmd.ProcessState == nil
}
```

测试文件 `manager_test.go` 顶部 import 加 `"os"`，并把 `osReadFile` 改为直接用 `os.ReadFile`：

```go
import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)
```

`readFile` 改为：

```go
func readFile(p string) (string, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
```

删除对 `osReadFile` 的引用。

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/frpc/ -v`
预期：PASS，4 个测试全过

> 注意：Windows 下 `ping -n 30 127.0.0.1` 是长进程；若 `exec.CommandContext` 在测试退出时未清理可能残留，`t.Cleanup` 已通过 `Stop` 间接处理。如偶发进程未终止，可在测试末尾加 `t.Cleanup(func() { _ = m.Stop(context.Background(), "s1") })`。

- [ ] **步骤 5：Commit**

```bash
git add client/internal/frpc/manager.go client/internal/frpc/manager_test.go
git commit -m "feat(client): frpc 进程管理启停重启 + 单测"
```

---

### 任务 5：App 方法接通真实依赖

**文件：**
- 修改：`client/app.go`、`client/types.go`、`client/main.go`

- [ ] **步骤 1：编写失败的测试**

`client/app_test.go`：

```go
package main

import (
	"path/filepath"
	"testing"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	a := &App{}
	dir := t.TempDir()
	if err := a.InitForTest(filepath.Join(dir, "test.db"), filepath.Join(dir, "frpc-cfg")); err != nil {
		t.Fatalf("InitForTest: %v", err)
	}
	return a
}

func TestAddAndListServers(t *testing.T) {
	a := newTestApp(t)
	id, err := a.AddServer(AddServerInput{
		Name: "prod", Host: "1.2.3.4", FrpsPort: 7000,
		FrpToken: "tok", AgentURL: "http://1.2.3.4:7400", AgentToken: "atok",
	})
	if err != nil {
		t.Fatalf("AddServer: %v", err)
	}
	if id == "" {
		t.Error("id 为空")
	}
	list, err := a.ListServers()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "prod" {
		t.Errorf("got %+v", list)
	}
}

func TestAddAndListTunnels(t *testing.T) {
	a := newTestApp(t)
	sid, _ := a.AddServer(AddServerInput{Name: "p", Host: "h", FrpsPort: 7000, FrpToken: "t", AgentURL: "u", AgentToken: "a"})
	tid, err := a.AddTunnel(AddTunnelInput{
		ServerID: sid, Name: "rdp", Protocol: "tcp",
		LocalIP: "127.0.0.1", LocalPort: 3389, RemotePort: 20389,
	})
	if err != nil {
		t.Fatalf("AddTunnel: %v", err)
	}
	if tid == "" {
		t.Error("tid 为空")
	}
	list, err := a.ListTunnels(sid)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "rdp" {
		t.Errorf("got %+v", list)
	}
}

func TestGenerateFrpcConfig(t *testing.T) {
	a := newTestApp(t)
	sid, _ := a.AddServer(AddServerInput{Name: "p", Host: "1.2.3.4", FrpsPort: 7000, FrpToken: "tok", AgentURL: "u", AgentToken: "a"})
	_, _ = a.AddTunnel(AddTunnelInput{ServerID: sid, Name: "rdp", Protocol: "tcp", LocalIP: "127.0.0.1", LocalPort: 3389, RemotePort: 20389})
	out, err := a.GenerateFrpcConfig(sid)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(out, `serverAddr = "1.2.3.4"`) {
		t.Errorf("缺少 serverAddr: %s", out)
	}
	if !strings.Contains(out, `remotePort = 20389`) {
		t.Errorf("缺少 remotePort: %s", out)
	}
}
```

测试 import 加 `"strings"`。

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./... -v -run TestAdd`
预期：FAIL，`undefined: App.InitForTest`、`undefined: AddServerInput`

- [ ] **步骤 3：编写实现**

`client/types.go` 末尾追加：

```go
// AddServerInput 是添加服务器的输入参数。
type AddServerInput struct {
	Name       string `json:"name"`
	Host       string `json:"host"`
	FrpsPort   int    `json:"frps_port"`
	FrpToken   string `json:"frp_token"`
	AgentURL   string `json:"agent_url"`
	AgentToken string `json:"agent_token"`
	IsDefault  bool   `json:"is_default"`
	Remark     string `json:"remark"`
}

// AddTunnelInput 是添加映射的输入参数。
type AddTunnelInput struct {
	ServerID     string `json:"server_id"`
	Name         string `json:"name"`
	Protocol     string `json:"protocol"`
	LocalIP      string `json:"local_ip"`
	LocalPort    int    `json:"local_port"`
	RemotePort   int    `json:"remote_port,omitempty"`
	CustomDomain string `json:"custom_domain,omitempty"`
	Subdomain    string `json:"subdomain,omitempty"`
}
```

`client/app.go`（整体替换）：

```go
package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kdc/frp-manager/client/internal/agent"
	"github.com/kdc/frp-manager/client/internal/db"
	"github.com/kdc/frp-manager/client/internal/frpc"
)

// App 是暴露给前端的 Wails 应用对象。
type App struct {
	ctx    context.Context
	repo   *db.Repo
	frpcMgr *frpc.Manager
}

// NewApp 创建 App 实例（依赖由 Init/InitForTest 注入）。
func NewApp() *App {
	return &App{}
}

// startup 在应用启动时由 Wails 调用。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Init 注入生产依赖（由 main.go 调用）。
func (a *App) Init(repo *db.Repo, frpcMgr *frpc.Manager) {
	a.repo = repo
	a.frpcMgr = frpcMgr
}

// InitForTest 注入测试依赖。
func (a *App) InitForTest(dbPath, frpcConfigDir string) error {
	d := db.Open(dbPath)
	r, err := db.NewRepo(d)
	if err != nil {
		return err
	}
	a.repo = r
	a.frpcMgr = frpc.NewManager(frpcConfigDir)
	return nil
}

func (a *App) newAgentClient(serverID string) (*agent.Client, error) {
	s, err := a.repo.GetServer(serverID)
	if err != nil {
		return nil, fmt.Errorf("服务器不存在: %w", err)
	}
	return agent.New(s.AgentURL, s.AgentToken), nil
}

// ListServers 返回所有服务器。
func (a *App) ListServers() ([]ServerInfo, error) {
	servers, err := a.repo.ListServers()
	if err != nil {
		return nil, err
	}
	out := make([]ServerInfo, 0, len(servers))
	for _, s := range servers {
		out = append(out, toServerInfo(s))
	}
	return out, nil
}

// AddServer 添加服务器，返回新 ID。
func (a *App) AddServer(in AddServerInput) (string, error) {
	now := time.Now().UTC()
	s := db.Server{
		ID: uuid.NewString(), Name: in.Name, Host: in.Host, FrpsPort: in.FrpsPort,
		FrpToken: in.FrpToken, AgentURL: in.AgentURL, AgentToken: in.AgentToken,
		IsDefault: in.IsDefault, Remark: in.Remark, CreatedAt: now, UpdatedAt: now,
	}
	if err := a.repo.InsertServer(s); err != nil {
		return "", err
	}
	return s.ID, nil
}

// UpdateServer 更新服务器。
func (a *App) UpdateServer(in AddServerInput) error {
	// 简化：通过 AgentURL 定位或要求前端传 ID；这里要求前端复用 AddServer 不合理，
	// 改为单独方法 UpdateServerByID。
	return fmt.Errorf("use UpdateServerByID")
}

// UpdateServerByID 按 ID 更新服务器。
func (a *App) UpdateServerByID(id string, in AddServerInput) error {
	now := time.Now().UTC()
	s := db.Server{
		ID: id, Name: in.Name, Host: in.Host, FrpsPort: in.FrpsPort,
		FrpToken: in.FrpToken, AgentURL: in.AgentURL, AgentToken: in.AgentToken,
		IsDefault: in.IsDefault, Remark: in.Remark, UpdatedAt: now,
	}
	return a.repo.UpdateServer(s)
}

// DeleteServer 删除服务器，并尝试通过 agent 释放其下映射占用的服务端资源。
func (a *App) DeleteServer(id string) error {
	tunnels, _ := a.repo.ListTunnelsByServer(id)
	if cli, err := a.newAgentClient(id); err == nil {
		for _, tu := range tunnels {
			a.releaseServerResource(cli, tu)
		}
	}
	return a.repo.DeleteServer(id)
}

// ListTunnels 返回指定服务器的映射；serverId 为空则返回全部。
func (a *App) ListTunnels(serverId string) ([]TunnelInfo, error) {
	if serverId == "" {
		all, err := a.repo.ListAllTunnels()
		if err != nil {
			return nil, err
		}
		out := make([]TunnelInfo, 0, len(all))
		for _, tu := range all {
			out = append(out, toTunnelInfo(tu))
		}
		return out, nil
	}
	list, err := a.repo.ListTunnelsByServer(serverId)
	if err != nil {
		return nil, err
	}
	out := make([]TunnelInfo, 0, len(list))
	for _, tu := range list {
		out = append(out, toTunnelInfo(tu))
	}
	return out, nil
}

// AddTunnel 添加映射，返回新 ID。
// 生产实现应在此调用 agent 校验/分配端口或域名，本计划先落库，计划 3 补全在线分配。
func (a *App) AddTunnel(in AddTunnelInput) (string, error) {
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
	return tu.ID, nil
}

// UpdateTunnelByID 按 ID 更新映射。
func (a *App) UpdateTunnelByID(id string, in AddTunnelInput) error {
	now := time.Now().UTC()
	tu := db.Tunnel{
		ID: id, ServerID: in.ServerID, Name: in.Name, Protocol: in.Protocol,
		LocalIP: in.LocalIP, LocalPort: in.LocalPort, RemotePort: in.RemotePort,
		CustomDomain: in.CustomDomain, Subdomain: in.Subdomain,
		Enabled: true, Status: "stopped", UpdatedAt: now,
	}
	return a.repo.UpdateTunnel(tu)
}

// DeleteTunnel 删除映射，并释放服务端资源。
func (a *App) DeleteTunnel(id string) error {
	tu, err := a.repo.GetTunnel(id)
	if err != nil {
		return fmt.Errorf("映射不存在: %w", err)
	}
	if cli, err := a.newAgentClient(tu.ServerID); err == nil {
		a.releaseServerResource(cli, *tu)
	}
	return a.repo.DeleteTunnel(id)
}

// releaseServerResource 根据协议释放端口或域名。
func (a *App) releaseServerResource(cli *agent.Client, tu db.Tunnel) {
	ctx := context.Background()
	switch tu.Protocol {
	case "tcp", "udp":
		if tu.RemotePort > 0 {
			_ = cli.ReleasePort(ctx, tu.Protocol, tu.RemotePort)
		}
	case "http", "https":
		domain := tu.CustomDomain
		if domain == "" && tu.Subdomain != "" {
			domain = tu.Subdomain
		}
		if domain != "" {
			_ = cli.ReleaseDomain(ctx, tu.Protocol, domain)
		}
	}
}

// GenerateFrpcConfig 根据指定服务器的映射生成 frpc.toml 内容。
func (a *App) GenerateFrpcConfig(serverId string) (string, error) {
	s, err := a.repo.GetServer(serverId)
	if err != nil {
		return "", fmt.Errorf("服务器不存在: %w", err)
	}
	tunnels, err := a.repo.ListTunnelsByServer(serverId)
	if err != nil {
		return "", err
	}
	cfg := &frpc.Config{
		ServerAddr: s.Host,
		ServerPort: s.FrpsPort,
		Auth:       frpc.Auth{Method: "token", Token: s.FrpToken},
	}
	for _, tu := range tunnels {
		if !tu.Enabled {
			continue
		}
		p := frpc.Proxy{
			Name: tu.Name, Type: tu.Protocol,
			LocalIP: tu.LocalIP, LocalPort: tu.LocalPort,
		}
		switch tu.Protocol {
		case "tcp", "udp":
			p.RemotePort = tu.RemotePort
		case "http", "https":
			if tu.CustomDomain != "" {
				p.CustomDomains = []string{tu.CustomDomain}
			} else if tu.Subdomain != "" {
				p.Subdomain = tu.Subdomain
			}
		}
		cfg.Proxies = append(cfg.Proxies, p)
	}
	return a.frpcMgr.Generate(cfg)
}

// StartFrpc 启动指定服务器的 frpc 进程。
func (a *App) StartFrpc(serverId string) error {
	cfgText, err := a.GenerateFrpcConfig(serverId)
	if err != nil {
		return err
	}
	return a.frpcMgr.Start(context.Background(), serverId, cfgText)
}

// StopFrpc 停止指定服务器的 frpc 进程。
func (a *App) StopFrpc(serverId string) error {
	return a.frpcMgr.Stop(context.Background(), serverId)
}

// RestartFrpc 重启指定服务器的 frpc 进程。
func (a *App) RestartFrpc(serverId string) error {
	cfgText, err := a.GenerateFrpcConfig(serverId)
	if err != nil {
		return err
	}
	return a.frpcMgr.Restart(context.Background(), serverId, cfgText)
}

// CheckServerCapabilities 查询服务端能力。
func (a *App) CheckServerCapabilities(serverId string) (*agent.Capabilities, error) {
	cli, err := a.newAgentClient(serverId)
	if err != nil {
		return nil, err
	}
	return cli.Capabilities(context.Background())
}

// IsFrpcRunning 返回指定服务器的 frpc 是否在运行。
func (a *App) IsFrpcRunning(serverId string) bool {
	return a.frpcMgr.IsRunning(serverId)
}

func toServerInfo(s db.Server) ServerInfo {
	return ServerInfo{
		ID: s.ID, Name: s.Name, Host: s.Host, FrpsPort: s.FrpsPort,
		FrpToken: s.FrpToken, AgentURL: s.AgentURL, AgentToken: s.AgentToken,
		IsDefault: s.IsDefault, Remark: s.Remark,
	}
}

func toTunnelInfo(tu db.Tunnel) TunnelInfo {
	return TunnelInfo{
		ID: tu.ID, ServerID: tu.ServerID, Name: tu.Name, Protocol: tu.Protocol,
		LocalIP: tu.LocalIP, LocalPort: tu.LocalPort, RemotePort: tu.RemotePort,
		CustomDomain: tu.CustomDomain, Subdomain: tu.Subdomain,
		Enabled: tu.Enabled, Status: tu.Status,
	}
}

var _ = strings.TrimSpace
```

- [ ] **步骤 4：更新 main.go 装配依赖**

`client/main.go` 在 `app := NewApp()` 后、`wails.Run` 前加入：

```go
	// 初始化客户端数据层与 frpc 管理
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
```

main.go import 块加（注意别名避免与 frpc/config 冲突）：

```go
	configpkg "github.com/kdc/frp-manager/client/internal/config"
	dbpkg "github.com/kdc/frp-manager/client/internal/db"
	"github.com/kdc/frp-manager/client/internal/frpc"
```

> 注意：原 `main.go` 的 `OnStartup: app.startup` 保留不动。

- [ ] **步骤 5：更新 go.mod 依赖**

运行：`go mod tidy`
预期：自动添加 `github.com/google/uuid`、`github.com/pelletier/go-toml/v2`

- [ ] **步骤 6：运行测试验证通过**

运行：`go test ./... -v`
预期：所有测试 PASS（含 app_test.go 三个）

- [ ] **步骤 7：编译验证**

运行：`go build ./... && go vet ./...`
预期：无输出

- [ ] **步骤 8：Commit**

```bash
git add client/app.go client/app_test.go client/types.go client/main.go client/go.mod client/go.sum
git commit -m "feat(client): App 接通真实依赖完成 v0.1 数据层闭环"
```

---

## 自检结果

**1. 规格覆盖度**（对照设计文档第 12、15、16.1、17、18 节）：
- 第 15 节 servers/tunnels 表 CRUD ✅ 任务 1
- 第 12 节 frpc.toml 生成（四协议）✅ 任务 3 + 任务 5 GenerateFrpcConfig
- 第 16.1 节服务器字段（含 agent_url/agent_token 必填）✅ 任务 5 AddServer
- 第 17 节创建流程的客户端落库部分 ✅ 任务 5 AddTunnel（在线校验/分配留计划 3）
- 第 18 节删除释放资源 ✅ 任务 5 DeleteTunnel/DeleteServer（调 agent release）
- 调用 server-agent 全部 8 个端点 ✅ 任务 2
- frpc 进程启停重启 ✅ 任务 4 + 任务 5

**2. 占位符扫描**：无 TODO/待定；每步有完整代码。`UpdateServer` 标注"use UpdateServerByID"是有意设计（ID 单独传），非占位。

**3. 类型一致性**：
- `db.Server`/`db.Tunnel` 全程字段名一致
- `agent.Capabilities` 字段与计划 1 server `CapabilitiesResponse` 一致（snake_case json tag）
- `frpc.Config` 嵌套 `Auth` 子结构，`Generate` 与 `App.GenerateFrpcConfig` 一致
- `frpc.Manager` 接口 `Start(ctx, serverID, cfgText)` 在任务 4 定义、任务 5 调用一致
- `App.Init(repo, frpcMgr)` 与 `main.go` 调用一致
- `AddServerInput`/`AddTunnelInput` 在 types.go 定义、app.go 使用、app_test.go 调用一致

**已识别的风险/注意事项：**
1. 任务 4 Windows 下进程替身用 `ping -n 30`，若测试退出未清理可能残留 ping 进程；已在测试末尾 `Stop` 并建议 `t.Cleanup`。
2. 任务 5 `main.go` import 用别名 `configpkg`/`dbpkg` 避免与 `frpc` 包内 `config` 概念冲突——但实际 `client/internal/config` 包名是 `config`，`frpc` 包内无 `config` 子包，别名可去掉。**建议实现时直接用 `config`、`db`、`frpc` 三个包名，无需别名**，除非编译报冲突。
3. `App.UpdateServer` 被设计为返回错误引导用 `UpdateServerByID`，略显冗余——实现时可删除 `UpdateServer` 方法只保留 `UpdateServerByID`。以实现时简洁为准。

这些都在对应任务给了处理方式，不阻塞。
