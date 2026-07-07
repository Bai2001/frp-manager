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

// ManagerIface 抽象域名操作（由 manager.go 中的 *Manager 结构体实现）。
// 保留接口便于未来 mock；当前所有使用方直接用 *Manager 结构体。
type ManagerIface interface {
	Check(ctx context.Context, p Protocol, domain string) (*CheckResult, error)
	Register(ctx context.Context, p Protocol, domain, tunnelID string) error
	Release(ctx context.Context, p Protocol, domain string) error
}
