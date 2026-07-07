package frpc

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// Manager 管理 frpc 进程，每个 serverID 对应一个进程 + 配置文件。
type Manager struct {
	mu           sync.Mutex
	procs        map[string]*exec.Cmd
	configDir    string
	binary       string
	args         []string
	appendConfig bool // 是否在 args 末尾追加配置文件路径（真实 frpc 用 -c <path>，测试替身无需追加）
}

// NewManager 创建进程管理器，configDir 用于存放每个 server 的 frpc.toml。
func NewManager(configDir string) *Manager {
	return &Manager{
		procs:        map[string]*exec.Cmd{},
		configDir:    configDir,
		binary:       "frpc",
		args:         []string{"-c"},
		appendConfig: true,
	}
}

// SetBinary 覆盖默认 frpc 二进制与参数（测试用替身）。
// 调用后不再自动追加配置文件路径到参数末尾，参数以传入的 args 为终态。
func (m *Manager) SetBinary(name string, args ...string) {
	m.binary = name
	m.args = args
	m.appendConfig = false
}

// Generate 委托给包级 Generate 函数。
func (m *Manager) Generate(cfg *Config) (string, error) {
	return Generate(cfg)
}

// Start 为指定 server 写入配置并启动 frpc。
func (m *Manager) Start(ctx context.Context, serverID, cfgText string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cmd, ok := m.procs[serverID]; ok && cmd.ProcessState == nil {
		return fmt.Errorf("server %s 的 frpc 已在运行", serverID)
	}
	if err := os.MkdirAll(m.configDir, 0o755); err != nil {
		return fmt.Errorf("创建配置目录: %w", err)
	}
	cfgPath := filepath.Join(m.configDir, serverID+".toml")
	if err := os.WriteFile(cfgPath, []byte(cfgText), 0o644); err != nil {
		return fmt.Errorf("写入 frpc 配置: %w", err)
	}
	args := append([]string{}, m.args...)
	if m.appendConfig {
		args = append(args, cfgPath)
	}
	cmd := exec.CommandContext(ctx, m.binary, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 frpc: %w", err)
	}
	m.procs[serverID] = cmd
	go func() { _ = cmd.Wait() }()
	return nil
}

// Stop 终止指定 server 的 frpc 进程。
func (m *Manager) Stop(_ context.Context, serverID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cmd, ok := m.procs[serverID]
	if !ok || cmd.Process == nil {
		return fmt.Errorf("server %s 未运行", serverID)
	}
	if err := cmd.Process.Kill(); err != nil {
		return fmt.Errorf("终止 frpc: %w", err)
	}
	delete(m.procs, serverID)
	return nil
}

// Restart 重启指定 server 的 frpc。
func (m *Manager) Restart(ctx context.Context, serverID, cfgText string) error {
	m.mu.Lock()
	if cmd, ok := m.procs[serverID]; ok && cmd.Process != nil {
		_ = cmd.Process.Kill()
		delete(m.procs, serverID)
	}
	m.mu.Unlock()
	return m.Start(ctx, serverID, cfgText)
}

// IsRunning 返回指定 server 的 frpc 是否在运行。
func (m *Manager) IsRunning(serverID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	cmd, ok := m.procs[serverID]
	if !ok {
		return false
	}
	return cmd.ProcessState == nil
}
