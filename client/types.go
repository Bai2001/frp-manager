package main

// ServerInfo 对应客户端 SQLite servers 表，前端通过 Wails 绑定接收。
// 字段使用 snake_case 以匹配前端 TS 接口（Wails 会保留 Go 字段名的 json tag 形式）。
type ServerInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Host       string `json:"host"`
	FrpsPort   int    `json:"frps_port"`
	FrpToken   string `json:"frp_token"`
	AgentURL   string `json:"agent_url"`
	AgentToken string `json:"agent_token"`
	IsDefault  bool   `json:"is_default"`
	Remark     string `json:"remark"`
}

// TunnelInfo 对应客户端 SQLite tunnels 表。
type TunnelInfo struct {
	ID           string `json:"id"`
	ServerID     string `json:"server_id"`
	Name         string `json:"name"`
	Protocol     string `json:"protocol"` // tcp | udp | http | https
	LocalIP      string `json:"local_ip"`
	LocalPort    int    `json:"local_port"`
	RemotePort   int    `json:"remote_port,omitempty"`
	CustomDomain string `json:"custom_domain,omitempty"`
	Subdomain    string `json:"subdomain,omitempty"`
	Enabled      bool   `json:"enabled"`
	Status       string `json:"status"`
}

// AddServerInput 是添加服务器的输入参数。
type AddServerInput struct {
	Name       string `json:"name"`
	Host       string `json:"host"`
	FrpsPort   int    `json:"frps_port"`
	FrpToken   string `json:"frp_token"`
	AgentURL   string `json:"agent_url"`
	AgentToken string `json:"agent_token"`
	IsDefault  bool   `json:"is_default"`
	Remark     string `json:"remark"`
}

// AddTunnelInput 是添加映射的输入参数。
type AddTunnelInput struct {
	ServerID     string `json:"server_id"`
	Name         string `json:"name"`
	Protocol     string `json:"protocol"`
	LocalIP      string `json:"local_ip"`
	LocalPort    int    `json:"local_port"`
	RemotePort   int    `json:"remote_port,omitempty"`
	CustomDomain string `json:"custom_domain,omitempty"`
	Subdomain    string `json:"subdomain,omitempty"`
}
