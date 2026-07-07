package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/kdc/frp-manager/client/internal/agent"
	"github.com/kdc/frp-manager/client/internal/config"
	"github.com/kdc/frp-manager/client/internal/db"
	"github.com/kdc/frp-manager/client/internal/frpc"
)

// App 是暴露给前端的 Wails 应用对象。
// 所有前端通过 v3 自动生成的 bindings 调用的方法都挂在 App 上。
type App struct {
	app      *application.App
	database *sql.DB
	repo     *db.Repo
	frpcMgr  *frpc.Manager
}

// NewApp 创建 App 实例（依赖由 Init/InitForTest 注入）。
func NewApp() *App {
	return &App{}
}

// SetApplication 注入 Wails v3 应用实例（由 main.go 在 application.New 后回填）。
// 供 EmitLog 等通过 app.Event.Emit 推送事件到前端。
func (a *App) SetApplication(app *application.App) {
	a.app = app
}

// ServiceStartup 在应用启动时由 Wails v3 调用。
// 初始化 db/repo/frpcMgr，返回 error 可中断启动。
func (a *App) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	dbPath, err := config.DefaultDBPath()
	if err != nil {
		return fmt.Errorf("获取默认 DB 路径: %w", err)
	}
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("打开客户端数据库: %w", err)
	}
	repo, err := db.NewRepo(database)
	if err != nil {
		_ = database.Close()
		return fmt.Errorf("初始化 db repo: %w", err)
	}
	a.database = database
	a.repo = repo
	a.frpcMgr = frpc.NewManager()
	a.frpcMgr.SetLogCallback(func(serverID, line string) {
		level := "info"
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") {
			level = "error"
		} else if strings.Contains(lower, "warn") {
			level = "warn"
		}
		a.EmitLog(level, line, serverID)
	})
	return nil
}

// ServiceShutdown 在应用退出时由 Wails v3 调用，停止所有 frpc 并释放数据库连接。
func (a *App) ServiceShutdown() error {
	var firstErr error
	if a.frpcMgr != nil {
		if err := a.frpcMgr.StopAll(); err != nil {
			firstErr = err
		}
	}
	if a.database != nil {
		if err := a.database.Close(); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// InitForTest 注入测试依赖，不经过 Wails 运行时。
func (a *App) InitForTest(dbPath string) error {
	d, err := db.Open(dbPath)
	if err != nil {
		return err
	}
	r, err := db.NewRepo(d)
	if err != nil {
		_ = d.Close()
		return err
	}
	a.database = d
	a.repo = r
	a.frpcMgr = frpc.NewManager()
	return nil
}

// Close 释放 App 持有的资源（供测试清理调用）。
// 生产环境由 ServiceShutdown 负责。
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
// 按 protocol 调用 agent 在线校验/分配端口或域名，失败则返回 error 不落库。
func (a *App) AddTunnel(in AddTunnelInput) (string, error) {
	ctx := context.Background()
	cli, err := a.newAgentClient(in.ServerID)
	if err != nil {
		return "", err
	}
	switch in.Protocol {
	case "tcp", "udp":
		if in.RemotePort > 0 {
			// 手动指定端口：检查可用性
			res, err := cli.CheckPort(ctx, in.Protocol, in.RemotePort)
			if err != nil {
				return "", fmt.Errorf("检查端口: %w", err)
			}
			if !res.Available {
				return "", fmt.Errorf("端口 %d 不可用: %s", in.RemotePort, res.Reason)
			}
		} else {
			// 自动分配
			port, err := cli.AllocatePort(ctx, in.Protocol)
			if err != nil {
				return "", fmt.Errorf("分配端口: %w", err)
			}
			in.RemotePort = port
		}
	case "http", "https":
		domain := in.CustomDomain
		if domain == "" {
			domain = in.Subdomain
		}
		if domain == "" {
			return "", fmt.Errorf("http/https 映射需提供 custom_domain 或 subdomain")
		}
		res, err := cli.CheckDomain(ctx, in.Protocol, domain)
		if err != nil {
			return "", fmt.Errorf("检查域名: %w", err)
		}
		if !res.Available {
			return "", fmt.Errorf("域名 %s 不可用: %s", domain, res.Reason)
		}
	}

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
	// 落库成功后注册域名占用（http/https）
	if in.Protocol == "http" || in.Protocol == "https" {
		domain := in.CustomDomain
		if domain == "" {
			domain = in.Subdomain
		}
		_ = cli.RegisterDomain(ctx, in.Protocol, domain, tu.ID)
	}
	return tu.ID, nil
}

// EmitLog 向前端推送一条日志（通过 Wails v3 事件 log:append）。
func (a *App) EmitLog(level, message, serverID string) {
	if a.app == nil {
		return
	}
	a.app.Event.Emit("log:append", map[string]string{
		"time":      time.Now().UTC().Format(time.RFC3339),
		"level":     level,
		"message":   message,
		"server_id": serverID,
	})
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

// buildFrpcConfig 根据 server 及其启用的 tunnel 构造 frpc 配置对象。
// 返回 common 与 proxies，供 GenerateFrpcConfig/StartFrpc/RestartFrpc 共用。
func (a *App) buildFrpcConfig(serverId string) (*v1.ClientCommonConfig, []v1.ProxyConfigurer, error) {
	s, err := a.repo.GetServer(serverId)
	if err != nil {
		return nil, nil, fmt.Errorf("服务器不存在: %w", err)
	}
	tunnels, err := a.repo.ListTunnelsByServer(serverId)
	if err != nil {
		return nil, nil, err
	}
	common := frpc.BuildClientConfig(s.Host, s.FrpsPort, s.FrpToken)
	var proxies []v1.ProxyConfigurer
	for _, tu := range tunnels {
		if !tu.Enabled {
			continue
		}
		p, err := frpc.BuildProxy(tu.Name, tu.Protocol, tu.LocalIP, tu.LocalPort, tu.RemotePort, tu.CustomDomain, tu.Subdomain)
		if err != nil {
			return nil, nil, err
		}
		proxies = append(proxies, p)
	}
	return common, proxies, nil
}

// GenerateFrpcConfig 根据指定服务器的映射生成 frpc.toml 内容。
func (a *App) GenerateFrpcConfig(serverId string) (string, error) {
	common, proxies, err := a.buildFrpcConfig(serverId)
	if err != nil {
		return "", err
	}
	return frpc.MarshalConfig(common, proxies)
}

// StartFrpc 启动指定服务器的 frpc 进程，并转发其输出为日志事件。
func (a *App) StartFrpc(serverId string) error {
	common, proxies, err := a.buildFrpcConfig(serverId)
	if err != nil {
		return err
	}
	if err := a.frpcMgr.Start(context.Background(), serverId, common, proxies); err != nil {
		a.EmitLog("error", "启动 frpc 失败: "+err.Error(), serverId)
		return err
	}
	a.EmitLog("info", "frpc 已启动", serverId)
	return nil
}

// StopFrpc 停止指定服务器的 frpc 进程。
func (a *App) StopFrpc(serverId string) error {
	return a.frpcMgr.Stop(context.Background(), serverId)
}

// RestartFrpc 重启指定服务器的 frpc 进程。
func (a *App) RestartFrpc(serverId string) error {
	common, proxies, err := a.buildFrpcConfig(serverId)
	if err != nil {
		return err
	}
	return a.frpcMgr.Restart(context.Background(), serverId, common, proxies)
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
		ID: tu.ID, ServerID: tu.ServerID, Name: tu.Name, Protocol: tu.Protocol,
		LocalIP: tu.LocalIP, LocalPort: tu.LocalPort, RemotePort: tu.RemotePort,
		CustomDomain: tu.CustomDomain, Subdomain: tu.Subdomain,
		Enabled: tu.Enabled, Status: tu.Status,
	}
}
