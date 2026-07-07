// Package agent 是调用 server-agent HTTP API 的客户端。
// 骨架阶段只定义能力检测相关类型，后续模块实现时填充。
package agent

import "context"

// Capabilities 描述服务端 frps 能力，对应 GET /api/capabilities 响应。
type Capabilities struct {
	FrpsRunning       bool   `json:"frps_running"`
	FrpsVersion       string `json:"frps_version"`
	BindPort          int    `json:"bind_port"`
	AllowPorts        []struct {
		Start int `json:"start"`
		End   int `json:"end"`
	} `json:"allow_ports"`
	SupportTCP    bool     `json:"support_tcp"`
	SupportUDP    bool     `json:"support_udp"`
	SupportHTTP   bool     `json:"support_http"`
	SupportHTTPS  bool     `json:"support_https"`
	VhostHTTPPort int      `json:"vhost_http_port"`
	VhostHTTPSPort int     `json:"vhost_https_port"`
	SubdomainHost  string  `json:"subdomain_host"`
	AllowedRootDomains []string `json:"allowed_root_domains"`
}

// Client 调用 server-agent API。
type Client struct {
	baseURL string
	token   string
}

// New 创建 agent 客户端。
func New(baseURL, token string) *Client {
	return &Client{baseURL: baseURL, token: token}
}

// Health 检查 agent 是否可达。
func (c *Client) Health(ctx context.Context) error {
	// TODO: 实现 GET /api/health
	return nil
}

// Capabilities 查询服务端能力。
func (c *Client) Capabilities(ctx context.Context) (*Capabilities, error) {
	// TODO: 实现 GET /api/capabilities
	return nil, nil
}
