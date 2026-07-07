// Package frpc 负责 frpc.toml 配置生成与 frpc 进程启停重启。
package frpc

import "context"

// Manager 抽象 frpc 的配置生成与进程控制（由 manager.go 实现）。
type Manager interface {
	Generate(cfg *Config) (string, error)
	Start(ctx context.Context, serverID string, cfgText string) error
	Stop(ctx context.Context, serverID string) error
	Restart(ctx context.Context, serverID string, cfgText string) error
}
