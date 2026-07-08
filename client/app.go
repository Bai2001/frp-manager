package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/kdc/frp-manager/client/internal/agent"
	"github.com/kdc/frp-manager/client/internal/autostart"
	"github.com/kdc/frp-manager/client/internal/config"
	"github.com/kdc/frp-manager/client/internal/db"
	"github.com/kdc/frp-manager/client/internal/frpc"
	"github.com/kdc/frp-manager/client/internal/logfile"
	"github.com/kdc/frp-manager/client/internal/settings"
)

// App 是暴露给前端的 Wails 应用对象。
// 所有前端通过 v3 自动生成的 bindings 调用的方法都挂在 App 上。
type App struct {
	app      *application.App
	database *sql.DB
	repo     *db.Repo
	frpcMgr  *frpc.Manager

	settingsStore *settings.Store
	logWriter     *logfile.Writer
	// settings 缓存，供托盘 WindowClosing hook 同步读取（避免每次关窗都读文件）。
	settings settings.Settings
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

// SetTray 注入系统托盘管理器（由 main.go 在 Tray.Setup 后回填）。
// 当前仅供未来扩展使用（如托盘菜单显示 frpc 运行状态）。
func (a *App) SetTray(t *Tray) {
	// 暂存引用，供后续状态联动扩展。当前无字段，预留接口。
	_ = t
}

// ServiceStartup 在应用启动时由 Wails v3 调用。
// 初始化 db/repo/frpcMgr/settings/logfile，返回 error 可中断启动。
func (a *App) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	a.EmitLog("info", "FRP Manager 启动中...", "")

	dbPath, err := config.DefaultDBPath()
	if err != nil {
		a.EmitLog("error", "获取默认 DB 路径失败: "+err.Error(), "")
		return fmt.Errorf("获取默认 DB 路径: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		a.EmitLog("error", "创建配置目录失败: "+err.Error(), "")
		return fmt.Errorf("创建配置目录: %w", err)
	}
	database, err := db.Open(dbPath)
	if err != nil {
		a.EmitLog("error", "打开数据库失败: "+err.Error(), "")
		return fmt.Errorf("打开客户端数据库: %w", err)
	}
	repo, err := db.NewRepo(database)
	if err != nil {
		_ = database.Close()
		a.EmitLog("error", "初始化数据库仓库失败: "+err.Error(), "")
		return fmt.Errorf("初始化 db repo: %w", err)
	}
	a.database = database
	a.repo = repo
	a.frpcMgr = frpc.NewManager()
	a.EmitLog("info", "数据库初始化完成", "")

	// 加载设置并初始化日志持久化
	cfgDir, _ := config.DefaultDir()
	a.settingsStore = settings.NewStore(filepath.Join(cfgDir, "settings.json"))
	a.settings, err = a.settingsStore.Load()
	if err != nil {
		// 设置损坏不阻塞启动，用零值继续
		a.EmitLog("warn", "设置文件读取失败，使用默认设置: "+err.Error(), "")
		a.settings = settings.Settings{}
	} else {
		a.EmitLog("info", fmt.Sprintf("设置已加载（日志保留 %d 天，开机自启 %v，关闭最小化到托盘 %v）",
			a.settings.LogRetentionDays, a.settings.AutoStart, a.settings.CloseToTray), "")
	}
	if a.settings.LogRetentionDays > 0 {
		a.logWriter = logfile.New(cfgDir, a.settings.LogRetentionDays)
		go a.logWriter.Cleanup() // 启动时清理一次过期日志
		a.EmitLog("info", fmt.Sprintf("日志持久化已启用，保留 %d 天", a.settings.LogRetentionDays), "")
	}

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

	a.EmitLog("info", "FRP Manager 启动完成", "")
	return nil
}

// ServiceShutdown 在应用退出时由 Wails v3 调用，停止所有 frpc 并释放资源。
func (a *App) ServiceShutdown() error {
	a.EmitLog("info", "FRP Manager 正在退出...", "")

	var firstErr error
	if a.frpcMgr != nil {
		a.EmitLog("info", "正在停止所有 frpc 进程...", "")
		if err := a.frpcMgr.StopAll(); err != nil {
			a.EmitLog("error", "停止 frpc 失败: "+err.Error(), "")
			firstErr = err
		} else {
			a.EmitLog("info", "所有 frpc 进程已停止", "")
		}
	}
	if a.logWriter != nil {
		if err := a.logWriter.Close(); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			a.EmitLog("warn", "关闭日志文件失败: "+err.Error(), "")
		}
	}
	if a.database != nil {
		if err := a.database.Close(); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			a.EmitLog("warn", "关闭数据库失败: "+err.Error(), "")
		}
	}

	if firstErr == nil {
		a.EmitLog("info", "FRP Manager 已安全退出", "")
	} else {
		a.EmitLog("error", "FRP Manager 退出时发生错误: "+firstErr.Error(), "")
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
	// 初始化 settingsStore（测试用临时目录），供 SaveSettings/GetSettings 测试
	a.settingsStore = settings.NewStore(filepath.Join(filepath.Dir(dbPath), "settings.json"))
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
		a.EmitLog("error", "添加服务器「"+in.Name+"」失败: "+err.Error(), "")
		return "", err
	}
	a.EmitLog("info", fmt.Sprintf("已添加服务器「%s」（%s:%d）", in.Name, in.Host, in.FrpsPort), "")
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
	if err := a.repo.UpdateServer(s); err != nil {
		a.EmitLog("error", "更新服务器「"+in.Name+"」失败: "+err.Error(), "")
		return err
	}
	a.EmitLog("info", "已更新服务器「"+in.Name+"」", "")
	return nil
}

// DeleteServer 删除服务器，并尝试通过 agent 释放其下映射占用的服务端资源。
func (a *App) DeleteServer(id string) error {
	tunnels, _ := a.repo.ListTunnelsByServer(id)
	serverName := id
	if s, err := a.repo.GetServer(id); err == nil {
		serverName = s.Name
	}
	if cli, err := a.newAgentClient(id); err == nil {
		for _, tu := range tunnels {
			a.releaseServerResource(cli, tu)
		}
	}
	if err := a.repo.DeleteServer(id); err != nil {
		a.EmitLog("error", "删除服务器「"+serverName+"」失败: "+err.Error(), "")
		return err
	}
	a.EmitLog("info", fmt.Sprintf("已删除服务器「%s」（含 %d 个映射）", serverName, len(tunnels)), "")
	return nil
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
		a.EmitLog("error", "创建映射「"+in.Name+"」失败: 无法连接 agent - "+err.Error(), in.ServerID)
		return "", err
	}
	switch in.Protocol {
	case "tcp", "udp":
		if in.RemotePort > 0 {
			// 手动指定端口：检查可用性
			a.EmitLog("info", fmt.Sprintf("正在检查端口 %d (%s) 是否可用...", in.RemotePort, in.Protocol), in.ServerID)
			res, err := cli.CheckPort(ctx, in.Protocol, in.RemotePort)
			if err != nil {
				a.EmitLog("error", fmt.Sprintf("检查端口 %d 失败: %s", in.RemotePort, err.Error()), in.ServerID)
				return "", fmt.Errorf("检查端口: %w", err)
			}
			if !res.Available {
				a.EmitLog("warn", fmt.Sprintf("端口 %d 不可用: %s", in.RemotePort, res.Reason), in.ServerID)
				return "", fmt.Errorf("端口 %d 不可用: %s", in.RemotePort, res.Reason)
			}
		} else {
			// 自动分配
			a.EmitLog("info", fmt.Sprintf("正在自动分配 %s 端口...", in.Protocol), in.ServerID)
			port, err := cli.AllocatePort(ctx, in.Protocol)
			if err != nil {
				a.EmitLog("error", "自动分配端口失败: "+err.Error(), in.ServerID)
				return "", fmt.Errorf("分配端口: %w", err)
			}
			in.RemotePort = port
			a.EmitLog("info", fmt.Sprintf("已分配端口 %d", port), in.ServerID)
		}
	case "http", "https":
		domain := in.CustomDomain
		if domain == "" {
			domain = in.Subdomain
		}
		if domain == "" {
			return "", fmt.Errorf("http/https 映射需提供 custom_domain 或 subdomain")
		}
		a.EmitLog("info", fmt.Sprintf("正在检查域名 %s (%s) 是否可用...", domain, in.Protocol), in.ServerID)
		res, err := cli.CheckDomain(ctx, in.Protocol, domain)
		if err != nil {
			a.EmitLog("error", fmt.Sprintf("检查域名 %s 失败: %s", domain, err.Error()), in.ServerID)
			return "", fmt.Errorf("检查域名: %w", err)
		}
		if !res.Available {
			a.EmitLog("warn", fmt.Sprintf("域名 %s 不可用: %s", domain, res.Reason), in.ServerID)
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
		a.EmitLog("error", "创建映射「"+in.Name+"」落库失败: "+err.Error(), in.ServerID)
		return "", err
	}
	a.EmitLog("info", fmt.Sprintf("已创建映射「%s」(%s → %s:%d)",
		in.Name, in.Protocol, in.LocalIP, in.LocalPort), in.ServerID)
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
// 同时按设置持久化到日志文件（若启用日志保留）。
func (a *App) EmitLog(level, message, serverID string) {
	if a.logWriter != nil {
		_ = a.logWriter.Write(level, message, serverID)
	}
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
		a.EmitLog("error", "删除映射失败: 映射不存在 - "+err.Error(), "")
		return fmt.Errorf("映射不存在: %w", err)
	}
	tunnelName := tu.Name
	if cli, err := a.newAgentClient(tu.ServerID); err == nil {
		a.releaseServerResource(cli, *tu)
	} else {
		a.EmitLog("warn", "释放服务端资源时无法连接 agent: "+err.Error(), tu.ServerID)
	}
	if err := a.repo.DeleteTunnel(id); err != nil {
		a.EmitLog("error", "删除映射「"+tunnelName+"」失败: "+err.Error(), tu.ServerID)
		return err
	}
	a.EmitLog("info", "已删除映射「"+tunnelName+"」", tu.ServerID)
	return nil
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
// 注意：Start 是异步的——立即返回成功仅表示 frpc service 已创建并开始尝试连接 frps，
// 真正是否连上要看日志事件和 IsFrpcRunning 轮询。连接失败会通过日志事件推送。
func (a *App) StartFrpc(serverId string) error {
	common, proxies, err := a.buildFrpcConfig(serverId)
	if err != nil {
		return err
	}
	if err := a.frpcMgr.Start(context.Background(), serverId, common, proxies); err != nil {
		a.EmitLog("error", "启动 frpc 失败: "+err.Error(), serverId)
		return err
	}
	a.EmitLog("info", "frpc 正在启动，等待连接 frps...", serverId)
	return nil
}

// StopFrpc 停止指定服务器的 frpc 进程。
func (a *App) StopFrpc(serverId string) error {
	a.EmitLog("info", "正在停止 frpc...", serverId)
	if err := a.frpcMgr.Stop(context.Background(), serverId); err != nil {
		a.EmitLog("error", "停止 frpc 失败: "+err.Error(), serverId)
		return err
	}
	a.EmitLog("info", "frpc 已停止", serverId)
	return nil
}

// RestartFrpc 重启指定服务器的 frpc 进程。
func (a *App) RestartFrpc(serverId string) error {
	common, proxies, err := a.buildFrpcConfig(serverId)
	if err != nil {
		a.EmitLog("error", "重启 frpc 失败: 构建配置出错 - "+err.Error(), serverId)
		return err
	}
	a.EmitLog("info", "正在重启 frpc...", serverId)
	if err := a.frpcMgr.Restart(context.Background(), serverId, common, proxies); err != nil {
		a.EmitLog("error", "重启 frpc 失败: "+err.Error(), serverId)
		return err
	}
	a.EmitLog("info", "frpc 已重启，等待连接 frps...", serverId)
	return nil
}

// CheckServerCapabilities 查询服务端能力。
func (a *App) CheckServerCapabilities(serverId string) (*agent.Capabilities, error) {
	a.EmitLog("info", "正在检测服务端能力...", serverId)
	cli, err := a.newAgentClient(serverId)
	if err != nil {
		a.EmitLog("error", "检测能力失败: 无法连接 agent - "+err.Error(), serverId)
		return nil, err
	}
	caps, err := cli.Capabilities(context.Background())
	if err != nil {
		a.EmitLog("error", "检测能力失败: "+err.Error(), serverId)
		return nil, err
	}
	a.EmitLog("info", fmt.Sprintf("服务端能力检测完成（frps %s，bind %d，支持 TCP=%v UDP=%v HTTP=%v HTTPS=%v）",
		caps.FrpsVersion, caps.BindPort, caps.SupportTCP, caps.SupportUDP, caps.SupportHTTP, caps.SupportHTTPS), serverId)
	return caps, nil
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

// ----- 设置 / 配置目录 / 导入导出 -----

// CloseToTray 返回当前是否启用"关闭最小化到托盘"。
// 供 Tray 的 WindowClosing hook 同步读取。
func (a *App) CloseToTray() bool {
	return a.settings.CloseToTray
}

// GetSettings 返回当前应用设置。设置未初始化时返回零值。
func (a *App) GetSettings() (settings.Settings, error) {
	if a.settingsStore == nil {
		return settings.Settings{}, nil
	}
	s, err := a.settingsStore.Load()
	if err != nil {
		return settings.Settings{}, fmt.Errorf("读取设置: %w", err)
	}
	a.settings = s // 刷新缓存
	return s, nil
}

// SaveSettings 保存设置并应用即时生效的部分（日志保留、开机自启）。
// 日志保留天数变化时重建 logWriter；开机自启变化时同步注册表/文件。
// 前端不感知窗口状态字段，此处保留服务端缓存的窗口状态，避免被零值覆盖。
func (a *App) SaveSettings(in settings.Settings) error {
	if a.settingsStore == nil {
		return fmt.Errorf("设置未初始化")
	}
	// 保留窗口状态字段（前端 Settings 接口不含这些字段，传入值为零值）
	in.WindowMaximised = a.settings.WindowMaximised
	in.WindowX = a.settings.WindowX
	in.WindowY = a.settings.WindowY
	in.WindowWidth = a.settings.WindowWidth
	in.WindowHeight = a.settings.WindowHeight
	// 开机自启：与当前状态不同时执行
	if in.AutoStart != a.settings.AutoStart {
		var err error
		if in.AutoStart {
			a.EmitLog("info", "正在启用开机自启...", "")
			err = autostart.Enable()
		} else {
			a.EmitLog("info", "正在关闭开机自启...", "")
			err = autostart.Disable()
		}
		if err != nil {
			a.EmitLog("error", "设置开机自启失败: "+err.Error(), "")
			return fmt.Errorf("设置开机自启: %w", err)
		}
		if in.AutoStart {
			a.EmitLog("info", "开机自启已启用", "")
		} else {
			a.EmitLog("info", "开机自启已关闭", "")
		}
	}
	// 日志保留天数变化：重建 logWriter
	if in.LogRetentionDays != a.settings.LogRetentionDays {
		if a.logWriter != nil {
			_ = a.logWriter.Close()
			a.logWriter = nil
		}
		if in.LogRetentionDays > 0 {
			cfgDir, _ := config.DefaultDir()
			a.logWriter = logfile.New(cfgDir, in.LogRetentionDays)
			go a.logWriter.Cleanup()
		}
		a.EmitLog("info", fmt.Sprintf("日志保留天数已更新为 %d 天", in.LogRetentionDays), "")
	}
	if err := a.settingsStore.Save(in); err != nil {
		a.EmitLog("error", "保存设置失败: "+err.Error(), "")
		return fmt.Errorf("保存设置: %w", err)
	}
	a.settings = in
	a.EmitLog("info", "设置已保存", "")
	return nil
}

// GetConfigDir 返回当前配置目录路径（只读展示用）。
func (a *App) GetConfigDir() string {
	dir, err := config.DefaultDir()
	if err != nil {
		return ""
	}
	return dir
}

// OpenConfigDir 用系统文件管理器打开配置目录。
func (a *App) OpenConfigDir() error {
	dir, err := config.DefaultDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", dir).Start()
	case "darwin":
		return exec.Command("open", dir).Start()
	default:
		return exec.Command("xdg-open", dir).Start()
	}
}

// BackupData 是导入导出的数据结构。
type BackupData struct {
	Servers    []db.Server `json:"servers"`
	Tunnels    []db.Tunnel `json:"tunnels"`
	ExportedAt time.Time   `json:"exported_at"`
	Version    string      `json:"version"`
}

// ExportData 导出所有服务器和映射为 JSON 字符串。
func (a *App) ExportData() (string, error) {
	if a.repo == nil {
		return "", fmt.Errorf("数据库未初始化")
	}
	servers, err := a.repo.ListServers()
	if err != nil {
		return "", fmt.Errorf("读取服务器: %w", err)
	}
	allTunnels, err := a.repo.ListAllTunnels()
	if err != nil {
		return "", fmt.Errorf("读取映射: %w", err)
	}
	backup := BackupData{
		Servers: servers, Tunnels: allTunnels,
		ExportedAt: time.Now().UTC(), Version: "0.2",
	}
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return "", err
	}
	a.EmitLog("info", fmt.Sprintf("已导出数据（%d 个服务器，%d 个映射）", len(servers), len(allTunnels)), "")
	return string(data), nil
}

// ImportData 从 JSON 字符串导入数据。策略：全量替换（先删后插）。
// 导入时不调用 agent 校验/注册，仅落库；用户后续启动 frpc 时若端口/域名冲突
// 会在 agent 校验阶段报错。
func (a *App) ImportData(raw string) error {
	if a.repo == nil {
		return fmt.Errorf("数据库未初始化")
	}
	var backup BackupData
	if err := json.Unmarshal([]byte(raw), &backup); err != nil {
		a.EmitLog("error", "导入数据解析失败: "+err.Error(), "")
		return fmt.Errorf("解析导入数据: %w", err)
	}
	a.EmitLog("info", fmt.Sprintf("开始导入数据（%d 个服务器，%d 个映射）", len(backup.Servers), len(backup.Tunnels)), "")
	// 全量替换：删除现有 tunnels 和 servers
	existingServers, err := a.repo.ListServers()
	if err != nil {
		return fmt.Errorf("读取现有服务器: %w", err)
	}
	for _, s := range existingServers {
		if err := a.repo.DeleteServer(s.ID); err != nil {
			a.EmitLog("error", "清理旧服务器失败: "+err.Error(), "")
			return fmt.Errorf("清理旧服务器 %s: %w", s.ID, err)
		}
	}
	// 插入导入的数据
	for _, s := range backup.Servers {
		if err := a.repo.InsertServer(s); err != nil {
			a.EmitLog("error", "插入服务器「"+s.Name+"」失败: "+err.Error(), "")
			return fmt.Errorf("插入服务器 %s: %w", s.Name, err)
		}
	}
	for _, tu := range backup.Tunnels {
		if err := a.repo.InsertTunnel(tu); err != nil {
			a.EmitLog("error", "插入映射「"+tu.Name+"」失败: "+err.Error(), "")
			return fmt.Errorf("插入映射 %s: %w", tu.Name, err)
		}
	}
	a.EmitLog("info", fmt.Sprintf("数据导入完成（%d 个服务器，%d 个映射）", len(backup.Servers), len(backup.Tunnels)), "")
	return nil
}
