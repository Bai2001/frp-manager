// Package domain 负责 HTTP/HTTPS 域名的校验、注册与释放。
// 骨架阶段只定义类型与接口，后续模块实现时填充。
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

// Manager 抽象域名操作。
type Manager interface {
	Check(ctx context.Context, p Protocol, domain string) (*CheckResult, error)
	Register(ctx context.Context, p Protocol, domain, tunnelID string) error
	Release(ctx context.Context, p Protocol, domain string) error
}

// noopManager 是骨架阶段的空实现。
type noopManager struct{}

// New 创建域名管理器占位实现。
func New() Manager { return &noopManager{} }

func (m *noopManager) Check(context.Context, Protocol, string) (*CheckResult, error) {
	return nil, nil
}
func (m *noopManager) Register(context.Context, Protocol, string, string) error { return nil }
func (m *noopManager) Release(context.Context, Protocol, string) error          { return nil }
