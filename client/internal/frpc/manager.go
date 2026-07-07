package frpc

import (
	"context"
	"fmt"
	"sync"

	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/pkg/config/source"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/policy/security"
)

// Manager 内嵌 frpc 服务，每个 serverID 对应一个 Service + ctx cancel。
// 用 generation 标记每次启动，避免 Restart 时旧 goroutine 退出误删新 service。
type Manager struct {
	mu          sync.Mutex
	services    map[string]*client.Service
	cancels     map[string]context.CancelFunc
	generations map[string]uint64
	dones       map[string]chan struct{}
	nextGen     uint64
	logCb       func(serverID, line string)
}

// NewManager 创建 frpc 管理器。
// 内嵌后不再需要 configDir（不写配置文件），故无参。
func NewManager() *Manager {
	return &Manager{
		services:    map[string]*client.Service{},
		cancels:     map[string]context.CancelFunc{},
		generations: map[string]uint64{},
		dones:       map[string]chan struct{}{},
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
	m.nextGen++
	gen := m.nextGen
	done := make(chan struct{})
	m.services[serverID] = svr
	m.cancels[serverID] = cancel
	m.generations[serverID] = gen
	go func() {
		_ = svr.Run(runCtx)
		m.mu.Lock()
		// 只在自己 generation 匹配时清理，避免 Restart 后旧 goroutine 误删新 service
		if m.generations[serverID] == gen {
			delete(m.services, serverID)
			delete(m.cancels, serverID)
			delete(m.generations, serverID)
		}
		m.mu.Unlock()
		close(done)
	}()
	// 记录 done 通道供 Restart 等待旧 service 退出
	if m.dones == nil {
		m.dones = map[string]chan struct{}{}
	}
	m.dones[serverID] = done
	return nil
}

// Stop 停止指定 server 的 frpc。
func (m *Manager) Stop(_ context.Context, serverID string) error {
	m.mu.Lock()
	cancel, ok := m.cancels[serverID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("server %s 未运行", serverID)
	}
	cancel()
	delete(m.services, serverID)
	delete(m.cancels, serverID)
	delete(m.generations, serverID)
	done := m.dones[serverID]
	delete(m.dones, serverID)
	m.mu.Unlock()
	// 等待旧 service goroutine 退出，确保端口/连接释放
	if done != nil {
		<-done
	}
	return nil
}

// Restart 重启指定 server 的 frpc。
// 先 Stop（等待旧 service 退出），再 Start，避免端口/连接冲突与状态竞态。
func (m *Manager) Restart(ctx context.Context, serverID string, common *v1.ClientCommonConfig, proxies []v1.ProxyConfigurer) error {
	if err := m.Stop(ctx, serverID); err != nil {
		// 未运行不算错误，继续 Start
		_ = err
	}
	return m.Start(ctx, serverID, common, proxies)
}

// IsRunning 返回指定 server 的 frpc 是否在运行。
func (m *Manager) IsRunning(serverID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.services[serverID]
	return ok
}

// StopAll 停止所有运行中的 frpc 服务，供应用退出时清理资源。
// 逐个调用 Stop 等待每个 service goroutine 退出，确保端口/连接释放。
func (m *Manager) StopAll() error {
	m.mu.Lock()
	ids := make([]string, 0, len(m.services))
	for id := range m.services {
		ids = append(ids, id)
	}
	m.mu.Unlock()

	var firstErr error
	for _, id := range ids {
		if err := m.Stop(context.Background(), id); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}
