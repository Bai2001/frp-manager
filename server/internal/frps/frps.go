// Package frps 负责 frps 进程管理、状态检测与配置读取。
// 骨架阶段只定义接口与空实现，后续模块实现时填充。
package frps

import "context"

// Status 描述 frps 当前状态。
type Status struct {
	Running   bool   `json:"running"`
	Version   string `json:"version,omitempty"`
	BindPort  int    `json:"bind_port,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Manager 抽象 frps 的控制能力。
type Manager interface {
	Status(ctx context.Context) (*Status, error)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Restart(ctx context.Context) error
	// ConfigPath 返回 frps.toml 路径，供配置读取/校验使用。
	ConfigPath() string
}

// noopManager 是骨架阶段的空实现。
type noopManager struct{ cfgPath string }

// New 创建一个 frps 管理器占位实现。
func New(cfgPath string) Manager {
	return &noopManager{cfgPath: cfgPath}
}

func (m *noopManager) Status(context.Context) (*Status, error) {
	return &Status{Running: false}, nil
}
func (m *noopManager) Start(context.Context) error    { return nil }
func (m *noopManager) Stop(context.Context) error     { return nil }
func (m *noopManager) Restart(context.Context) error  { return nil }
func (m *noopManager) ConfigPath() string             { return m.cfgPath }
