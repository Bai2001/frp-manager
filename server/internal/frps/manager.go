package frps

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/kdc/frp-manager/server/internal/frpsc"
)

// Manager 是 frps 状态检测的真实实现。
// 通过尝试连接 frps bindPort 判断是否运行，通过 frpsc.Parse 解析配置。
type Manager struct {
	cfgPath    string
	binaryPath string
}

// NewManager 创建 frps 管理器。binaryPath 可为空（骨架阶段不启动真实进程）。
func NewManager(cfgPath, binaryPath string) *Manager {
	return &Manager{cfgPath: cfgPath, binaryPath: binaryPath}
}

// Status 检测 frps 是否运行并解析其配置。
func (m *Manager) Status(_ context.Context) (*Status, error) {
	cfg, err := frpsc.Parse(m.cfgPath)
	if err != nil {
		return nil, fmt.Errorf("解析 frps 配置: %w", err)
	}
	running := isPortListening(cfg.BindPort)
	return &Status{
		Running:  running,
		BindPort: cfg.BindPort,
	}, nil
}

// Config 返回解析后的 frps 配置。
func (m *Manager) Config() (*frpsc.Config, error) {
	return frpsc.Parse(m.cfgPath)
}

// ConfigPath 返回 frps.toml 路径。
func (m *Manager) ConfigPath() string { return m.cfgPath }

// Start/Stop/Restart 在 v0.1 暂不实现真实进程管理（设计文档将其归入 v0.3），
// 但保留方法以满足接口，返回 not implemented。
func (m *Manager) Start(context.Context) error   { return errNotImpl }
func (m *Manager) Stop(context.Context) error    { return errNotImpl }
func (m *Manager) Restart(context.Context) error { return errNotImpl }

var errNotImpl = fmt.Errorf("frps 进程管理在 v0.1 未实现")

// isPortListening 判断本地 TCP 端口是否在监听。
func isPortListening(port int) bool {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:"+strconv.Itoa(port), 300*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
