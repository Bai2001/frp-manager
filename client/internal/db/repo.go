package db

import (
	"database/sql"
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
