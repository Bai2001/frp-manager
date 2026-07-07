package main

import (
	"context"
)

// App 是暴露给前端的 Wails 应用对象。
// 所有前端通过 window.go 调用的方法都挂在 App 上。
// 骨架阶段方法返回空值，待 internal/agent、internal/db、internal/frpc 实现后接入。
type App struct {
	ctx context.Context
}

// NewApp 创建 App 实例。
func NewApp() *App {
	return &App{}
}

// startup 在应用启动时由 Wails 调用，保存上下文以便调用 runtime。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// ListServers 返回所有已配置的服务器。
func (a *App) ListServers() []ServerInfo {
	// TODO: 接 internal/db
	return []ServerInfo{}
}

// ListTunnels 返回指定服务器的映射列表；serverId 为空则返回全部。
func (a *App) ListTunnels(serverId string) []TunnelInfo {
	// TODO: 接 internal/db
	return []TunnelInfo{}
}

// GenerateFrpcConfig 根据指定服务器的映射生成 frpc.toml 内容。
func (a *App) GenerateFrpcConfig(serverId string) (string, error) {
	// TODO: 接 internal/frpc
	return "", nil
}

// StartFrpc 启动指定服务器的 frpc 进程。
func (a *App) StartFrpc(serverId string) error {
	// TODO: 接 internal/frpc
	return nil
}

// StopFrpc 停止指定服务器的 frpc 进程。
func (a *App) StopFrpc(serverId string) error {
	// TODO: 接 internal/frpc
	return nil
}

// RestartFrpc 重启指定服务器的 frpc 进程。
func (a *App) RestartFrpc(serverId string) error {
	// TODO: 接 internal/frpc
	return nil
}
