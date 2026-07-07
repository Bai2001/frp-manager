// Package frpc 负责 frpc.toml 配置生成与 frpc 进程启停重启。
// 骨架阶段只定义类型与接口，后续模块实现时填充。
package frpc

import "context"

// Proxy 对应 frpc.toml 中的一个 [[proxies]] 段。
type Proxy struct {
	Name          string   `toml:"name"`
	Type          string   `toml:"type"` // tcp | udp | http | https
	LocalIP       string   `toml:"localIP"`
	LocalPort     int      `toml:"localPort"`
	RemotePort    int      `toml:"remotePort,omitempty"`
	CustomDomains []string `toml:"customDomains,omitempty"`
	Subdomain     string   `toml:"subdomain,omitempty"`
}

// Config 是 frpc.toml 的完整结构。
type Config struct {
	ServerAddr string  `toml:"serverAddr"`
	ServerPort int     `toml:"serverPort"`
	AuthToken  string  `toml:"auth.token,omitempty"`
	Proxies    []Proxy `toml:"proxies"`
}

// Manager 抽象 frpc 的配置生成与进程控制。
type Manager interface {
	Generate(cfg *Config) (string, error)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Restart(ctx context.Context) error
}

// noopManager 是骨架阶段的空实现。
type noopManager struct{}

// New 创建 frpc 管理器占位实现。
func New() Manager { return &noopManager{} }

func (m *noopManager) Generate(*Config) (string, error) { return "", nil }
func (m *noopManager) Start(context.Context) error      { return nil }
func (m *noopManager) Stop(context.Context) error       { return nil }
func (m *noopManager) Restart(context.Context) error    { return nil }
