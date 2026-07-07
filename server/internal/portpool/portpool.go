// Package portpool 负责 TCP/UDP 远程端口的检测、分配与释放。
// 骨架阶段只定义类型与接口，后续模块实现时填充。
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

// Manager 抽象端口池操作。
type Manager interface {
	Check(ctx context.Context, p Protocol, port int) (*CheckResult, error)
	Allocate(ctx context.Context, p Protocol) (int, error)
	Release(ctx context.Context, p Protocol, port int) error
}

// noopManager 是骨架阶段的空实现。
type noopManager struct{}

// New 创建端口池管理器占位实现。
func New() Manager { return &noopManager{} }

func (m *noopManager) Check(context.Context, Protocol, int) (*CheckResult, error) {
	return nil, nil
}
func (m *noopManager) Allocate(context.Context, Protocol) (int, error) { return 0, nil }
func (m *noopManager) Release(context.Context, Protocol, int) error    { return nil }
