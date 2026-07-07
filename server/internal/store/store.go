// Package store 负责 server-agent 的 SQLite 连接与 migration 执行。
package store

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/0001_init.sql
var initSQL string

// Open 打开指定路径的 SQLite 数据库并执行初始化 migration。
// 若父目录不存在会自动创建。
func Open(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("创建数据库目录: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开 sqlite: %w", err)
	}
	if _, err := db.Exec(initSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("执行初始化 migration: %w", err)
	}
	return db, nil
}

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
