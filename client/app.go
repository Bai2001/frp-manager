package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/kdc/frp-manager/client/internal/agent"
	"github.com/kdc/frp-manager/client/internal/db"
	"github.com/kdc/frp-manager/client/internal/frpc"
)

// App 是暴露给前端的 Wails 应用对象。
// 所有前端通过 window.go 调用的方法都挂在 App 上。
type App struct {
	ctx      context.Context
	database *sql.DB
	repo     *db.Repo
	frpcMgr  *frpc.Manager
}

// NewApp 创建 App 实例（依赖由 Init/InitForTest 注入）。
func NewApp() *App {
	return &App{}
}

// startup 在应用启动时由 Wails 调用。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Init 注入生产依赖（由 main.go 调用）。
func (a *App) Init(repo *db.Repo, frpcMgr *frpc.Manager) {
	a.repo = repo
	a.frpcMgr = frpcMgr
}

// SetDatabase 注入生产环境的底层数据库连接（供退出时关闭）。
func (a *App) SetDatabase(d *sql.DB) {
	a.database = d
}

// InitForTest 注入测试依赖。
func (a *App) InitForTest(dbPath, frpcConfigDir string) error {
	d, err := db.Open(dbPath)
	if err != nil {
		return err
	}
	r, err := db.NewRepo(d)
	if err != nil {
		return err
	}
	a.database = d
	a.repo = r
	a.frpcMgr = frpc.NewManager(frpcConfigDir)
	return nil
}

// Close 释放 App 持有的资源（如数据库连接），供测试清理或应用退出调用。
func (a *App) Close() {
	if a.database != nil {
		_ = a.database.Close()
	}
}

func (a *App) newAgentClient(serverID string) (*agent.Client, error) {
	s, err := a.repo.GetServer(serverID)
	if err != nil {
		return nil, fmt.Errorf("服务器不存在: %w", err)
	}
	return agent.New(s.AgentURL, s.AgentToken), nil
}

// ListServers 返回所有服务器。
func (a *App) ListServers() ([]ServerInfo, error) {
	servers, err := a.repo.ListServers()
	if err != nil {
		return nil, err
	}
	out := make([]ServerInfo, 0, len(servers))
	for _, s := range servers {
		out = append(out, toServerInfo(s))
	}
	return out, nil
}

// AddServer 添加服务器，返回新 ID。
func (a *App) AddServer(in AddServerInput) (string, error) {
	now := time.Now().UTC()
	s := db.Server{
		ID: uuid.NewString(), Name: in.Name, Host: in.Host, FrpsPort: in.FrpsPort,
		FrpToken: in.FrpToken, AgentURL: in.AgentURL, AgentToken: in.AgentToken,
		IsDefault: in.IsDefault, Remark: in.Remark, CreatedAt: now, UpdatedAt: now,
	}
	if err := a.repo.InsertServer(s); err != nil {
		return "", err
	}
	return s.ID, nil
}

// UpdateServerByID 按 ID 更新服务器。
func (a *App) UpdateServerByID(id string, in AddServerInput) error {
	now := time.Now().UTC()
	s := db.Server{
		ID: id, Name: in.Name, Host: in.Host, FrpsPort: in.FrpsPort,
		FrpToken: in.FrpToken, AgentURL: in.AgentURL, AgentToken: in.AgentToken,
		IsDefault: in.IsDefault, Remark: in.Remark, UpdatedAt: now,
	}
	return a.repo.UpdateServer(s)
}

// DeleteServer 删除服务器，并尝试通过 agent 释放其下映射占用的服务端资源。
func (a *App) DeleteServer(id string) error {
	tunnels, _ := a.repo.ListTunnelsByServer(id)
	if cli, err := a.newAgentClient(id); err == nil {
		for _, tu := range tunnels {
			a.releaseServerResource(cli, tu)
		}
	}
	return a.repo.DeleteServer(id)
}

// ListTunnels 返回指定服务器的映射；serverId 为空则返回全部。
func (a *App) ListTunnels(serverId string) ([]TunnelInfo, error) {
	if serverId == "" {
		all, err := a.repo.ListAllTunnels()
		if err != nil {
			return nil, err
		}
		out := make([]TunnelInfo, 0, len(all))
		for _, tu := range all {
			out = append(out, toTunnelInfo(tu))
		}
		return out, nil
	}
	list, err := a.repo.ListTunnelsByServer(serverId)
	if err != nil {
		return nil, err
	}
	out := make([]TunnelInfo, 0, len(list))
	for _, tu := range list {
		out = append(out, toTunnelInfo(tu))
	}
	return out, nil
}

// AddTunnel 添加映射，返回新 ID。
// 生产实现应在此调用 agent 校验/分配端口或域名，本计划先落库，计划 3 补全在线分配。
func (a *App) AddTunnel(in AddTunnelInput) (string, error) {
	now := time.Now().UTC()
	tu := db.Tunnel{
		ID: uuid.NewString(), ServerID: in.ServerID, Name: in.Name, Protocol: in.Protocol,
		LocalIP: in.LocalIP, LocalPort: in.LocalPort, RemotePort: in.RemotePort,
		CustomDomain: in.CustomDomain, Subdomain: in.Subdomain,
		Enabled: true, Status: "stopped", CreatedAt: now, UpdatedAt: now,
	}
	if err := a.repo.InsertTunnel(tu); err != nil {
		return "", err
	}
	return tu.ID, nil
}

// UpdateTunnelByID 按 ID 更新映射。
func (a *App) UpdateTunnelByID(id string, in AddTunnelInput) error {
	now := time.Now().UTC()
	tu := db.Tunnel{
		ID: id, ServerID: in.ServerID, Name: in.Name, Protocol: in.Protocol,
		LocalIP: in.LocalIP, LocalPort: in.LocalPort, RemotePort: in.RemotePort,
		CustomDomain: in.CustomDomain, Subdomain: in.Subdomain,
		Enabled: true, Status: "stopped", UpdatedAt: now,
	}
	return a.repo.UpdateTunnel(tu)
}

// DeleteTunnel 删除映射，并释放服务端资源。
func (a *App) DeleteTunnel(id string) error {
	tu, err := a.repo.GetTunnel(id)
	if err != nil {
		return fmt.Errorf("映射不存在: %w", err)
	}
	if cli, err := a.newAgentClient(tu.ServerID); err == nil {
		a.releaseServerResource(cli, *tu)
	}
	return a.repo.DeleteTunnel(id)
}

// releaseServerResource 根据协议释放端口或域名。
func (a *App) releaseServerResource(cli *agent.Client, tu db.Tunnel) {
	ctx := context.Background()
	switch tu.Protocol {
	case "tcp", "udp":
		if tu.RemotePort > 0 {
			_ = cli.ReleasePort(ctx, tu.Protocol, tu.RemotePort)
		}
	case "http", "https":
		domain := tu.CustomDomain
		if domain == "" && tu.Subdomain != "" {
			domain = tu.Subdomain
		}
		if domain != "" {
			_ = cli.ReleaseDomain(ctx, tu.Protocol, domain)
		}
	}
}

// GenerateFrpcConfig 根据指定服务器的映射生成 frpc.toml 内容。
func (a *App) GenerateFrpcConfig(serverId string) (string, error) {
	s, err := a.repo.GetServer(serverId)
	if err != nil {
		return "", fmt.Errorf("服务器不存在: %w", err)
	}
	tunnels, err := a.repo.ListTunnelsByServer(serverId)
	if err != nil {
		return "", err
	}
	cfg := &frpc.Config{
		ServerAddr: s.Host,
		ServerPort: s.FrpsPort,
		Auth:       frpc.Auth{Method: "token", Token: s.FrpToken},
	}
	for _, tu := range tunnels {
		if !tu.Enabled {
			continue
		}
		p := frpc.Proxy{
			Name: tu.Name, Type: tu.Protocol,
			LocalIP: tu.LocalIP, LocalPort: tu.LocalPort,
		}
		switch tu.Protocol {
		case "tcp", "udp":
			p.RemotePort = tu.RemotePort
		case "http", "https":
			if tu.CustomDomain != "" {
				p.CustomDomains = []string{tu.CustomDomain}
			} else if tu.Subdomain != "" {
				p.Subdomain = tu.Subdomain
			}
		}
		cfg.Proxies = append(cfg.Proxies, p)
	}
	return a.frpcMgr.Generate(cfg)
}

// StartFrpc 启动指定服务器的 frpc 进程。
func (a *App) StartFrpc(serverId string) error {
	cfgText, err := a.GenerateFrpcConfig(serverId)
	if err != nil {
		return err
	}
	return a.frpcMgr.Start(context.Background(), serverId, cfgText)
}

// StopFrpc 停止指定服务器的 frpc 进程。
func (a *App) StopFrpc(serverId string) error {
	return a.frpcMgr.Stop(context.Background(), serverId)
}

// RestartFrpc 重启指定服务器的 frpc 进程。
func (a *App) RestartFrpc(serverId string) error {
	cfgText, err := a.GenerateFrpcConfig(serverId)
	if err != nil {
		return err
	}
	return a.frpcMgr.Restart(context.Background(), serverId, cfgText)
}

// CheckServerCapabilities 查询服务端能力。
func (a *App) CheckServerCapabilities(serverId string) (*agent.Capabilities, error) {
	cli, err := a.newAgentClient(serverId)
	if err != nil {
		return nil, err
	}
	return cli.Capabilities(context.Background())
}

// IsFrpcRunning 返回指定服务器的 frpc 是否在运行。
func (a *App) IsFrpcRunning(serverId string) bool {
	return a.frpcMgr.IsRunning(serverId)
}

func toServerInfo(s db.Server) ServerInfo {
	return ServerInfo{
		ID: s.ID, Name: s.Name, Host: s.Host, FrpsPort: s.FrpsPort,
		FrpToken: s.FrpToken, AgentURL: s.AgentURL, AgentToken: s.AgentToken,
		IsDefault: s.IsDefault, Remark: s.Remark,
	}
}

func toTunnelInfo(tu db.Tunnel) TunnelInfo {
	return TunnelInfo{
		ID:       tu.ID, ServerID: tu.ServerID, Name: tu.Name, Protocol: tu.Protocol,
		LocalIP:   tu.LocalIP, LocalPort: tu.LocalPort, RemotePort: tu.RemotePort,
		CustomDomain: tu.CustomDomain, Subdomain: tu.Subdomain,
		Enabled:   tu.Enabled, Status: tu.Status,
	}
}
