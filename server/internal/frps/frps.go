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

// ManagerIface 抽象 frps 的控制能力（由 manager.go 中的 *Manager 结构体实现）。
// 保留接口便于未来 mock；当前所有使用方直接用 *Manager 结构体。
type ManagerIface interface {
	Status(ctx context.Context) (*Status, error)
	Config() (*frpsc.Config, error)
	ConfigPath() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Restart(ctx context.Context) error
}
