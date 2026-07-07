package frps

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/fatedier/frp/pkg/config"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/util/version"
	"github.com/fatedier/frp/server"
)

// Manager 内嵌 frps 服务。
type Manager struct {
	mu       sync.Mutex
	cfgPath  string
	cancel   context.CancelFunc
	running  bool
	bindPort int
}

// NewManager 创建 frps 管理器。
// 内嵌模式下不再需要 frps 二进制路径，只需 frps.toml 配置路径。
func NewManager(cfgPath string) *Manager {
	return &Manager{cfgPath: cfgPath}
}

// Config 加载并返回 frps 配置（官方 v1.ServerConfig）。
func (m *Manager) Config() (*v1.ServerConfig, error) {
	cfg, _, err := config.LoadServerConfig(m.cfgPath, true)
	if err != nil {
		return nil, fmt.Errorf("加载 frps 配置 %s: %w", m.cfgPath, err)
	}
	return cfg, nil
}

// ConfigPath 返回 frps.toml 路径。
func (m *Manager) ConfigPath() string { return m.cfgPath }

// Status 检测 frps 是否运行。
// 若内嵌 frps 已启动，以运行状态为准；否则回退到端口探测。
func (m *Manager) Status(_ context.Context) (*Status, error) {
	cfg, err := m.Config()
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	running := m.running
	m.mu.Unlock()
	if !running {
		running = isPortListening(cfg.BindPort)
	}
	return &Status{
		Running:  running,
		Version:  version.Full(),
		BindPort: cfg.BindPort,
	}, nil
}

// Start 内嵌启动 frps。
// 通过 server.NewService 构造 frps 服务，goroutine 中运行 svr.Run(ctx)。
// ctx cancel 即停止 frps。
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running {
		return fmt.Errorf("frps 已在运行")
	}
	cfg, err := m.Config()
	if err != nil {
		return err
	}
	svr, err := server.NewService(cfg)
	if err != nil {
		return fmt.Errorf("创建 frps service: %w", err)
	}
	runCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.running = true
	m.bindPort = cfg.BindPort
	go func() {
		svr.Run(runCtx)
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
	}()
	// 给 frps 一点启动时间绑定端口
	time.Sleep(200 * time.Millisecond)
	return nil
}

// Stop 停止内嵌 frps（通过 ctx cancel）。
func (m *Manager) Stop(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running || m.cancel == nil {
		return fmt.Errorf("frps 未运行")
	}
	m.cancel()
	m.cancel = nil
	m.running = false
	return nil
}

// Restart 重启 frps。
func (m *Manager) Restart(ctx context.Context) error {
	_ = m.Stop(ctx)
	return m.Start(ctx)
}

// isPortListening 判断本地 TCP 端口是否在监听。
func isPortListening(port int) bool {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:"+strconv.Itoa(port), 300*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
