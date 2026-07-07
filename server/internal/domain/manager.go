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

// isValidDomain 简单校验域名格式：非空、只含合法字符、长度合法。
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
