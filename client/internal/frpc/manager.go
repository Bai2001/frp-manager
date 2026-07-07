package frpc

import (
	"context"
	"fmt"
	"sync"

	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/pkg/config/source"
	"github.com/fatedier/frp/pkg/policy/security"
	v1 "github.com/fatedier/frp/pkg/config/v1"
)

// Manager 内嵌 frpc 服务，每个 serverID 对应一个 Service + ctx cancel。
type Manager struct {
	mu       sync.Mutex
	services map[string]*client.Service
	cancels  map[string]context.CancelFunc
	logCb    func(serverID, line string)
}

// NewManager 创建 frpc 管理器。
// 内嵌后不再需要 configDir（不写配置文件），故无参。
func NewManager() *Manager {
	return &Manager{
		services: map[string]*client.Service{},
		cancels:  map[string]context.CancelFunc{},
	}
}

// SetLogCallback 设置日志回调。
// 注意：frpc 内嵌后日志走 frp 全局 logger（pkg/util/log），v0.1 暂不深度集成,
// 此回调保留接口但不会实际捕获 frp 内部日志。
func (m *Manager) SetLogCallback(cb func(serverID, line string)) {
	m.logCb = cb
}

// Start 为指定 server 内嵌启动 frpc。
// common 会先调用 Complete() 完善默认值，proxies 通过 config source 注入 Service。
func (m *Manager) Start(ctx context.Context, serverID string, common *v1.ClientCommonConfig, proxies []v1.ProxyConfigurer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.services[serverID]; ok {
		return fmt.Errorf("server %s 的 frpc 已在运行", serverID)
	}
	if err := common.Complete(); err != nil {
		return fmt.Errorf("完善 client config: %w", err)
	}
	configSource := source.NewConfigSource()
	if err := configSource.ReplaceAll(proxies, nil); err != nil {
		return fmt.Errorf("设置 config source: %w", err)
	}
	aggregator := source.NewAggregator(configSource)
	unsafe := security.NewUnsafeFeatures(nil)
	svr, err := client.NewService(client.ServiceOptions{
		Common:                 common,
		ConfigSourceAggregator: aggregator,
		UnsafeFeatures:         unsafe,
	})
	if err != nil {
		return fmt.Errorf("创建 frpc service: %w", err)
	}
	runCtx, cancel := context.WithCancel(ctx)
	m.services[serverID] = svr
	m.cancels[serverID] = cancel
	go func() {
		_ = svr.Run(runCtx)
		m.mu.Lock()
		delete(m.services, serverID)
		delete(m.cancels, serverID)
		m.mu.Unlock()
	}()
	return nil
}

// Stop 停止指定 server 的 frpc。
func (m *Manager) Stop(_ context.Context, serverID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cancel, ok := m.cancels[serverID]
	if !ok {
		return fmt.Errorf("server %s 未运行", serverID)
	}
	cancel()
	delete(m.services, serverID)
	delete(m.cancels, serverID)
	return nil
}

// Restart 重启指定 server 的 frpc。
func (m *Manager) Restart(ctx context.Context, serverID string, common *v1.ClientCommonConfig, proxies []v1.ProxyConfigurer) error {
	_ = m.Stop(ctx, serverID)
	return m.Start(ctx, serverID, common, proxies)
}

// IsRunning 返回指定 server 的 frpc 是否在运行。
func (m *Manager) IsRunning(serverID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.services[serverID]
	return ok
}
