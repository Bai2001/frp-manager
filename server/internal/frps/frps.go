// Package frps 内嵌 frps 服务并管理其生命周期。
package frps

import (
	"context"

	v1 "github.com/fatedier/frp/pkg/config/v1"
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
	Config() (*v1.ServerConfig, error)
	ConfigPath() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Restart(ctx context.Context) error
}
