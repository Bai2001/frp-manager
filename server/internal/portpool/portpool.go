// Package portpool 负责 TCP/UDP 远程端口的检测、分配与释放。
package portpool

import "context"

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

// ManagerIface 抽象端口池操作（由 manager.go 中的 *Manager 结构体实现）。
// 保留接口便于未来 mock；当前所有使用方直接用 *Manager 结构体。
type ManagerIface interface {
	Check(ctx context.Context, p Protocol, port int) (*CheckResult, error)
	Allocate(ctx context.Context, p Protocol) (int, error)
	Release(ctx context.Context, p Protocol, port int) error
}
