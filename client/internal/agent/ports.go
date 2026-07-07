package agent

import (
	"context"
	"net/http"
	"strconv"
)

// PortCheckResult 对应 GET /api/ports/check 响应。
type PortCheckResult struct {
	Protocol  string `json:"protocol"`
	Port      int    `json:"port"`
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
}

// PortAllocateResult 对应 POST /api/ports/allocate 响应。
type PortAllocateResult struct {
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
}

// CheckPort 检查端口可用性。
func (c *Client) CheckPort(ctx context.Context, protocol string, port int) (*PortCheckResult, error) {
	var res PortCheckResult
	path := c.buildURL("/api/ports/check", map[string]string{"protocol": protocol, "port": strconv.Itoa(port)})
	if err := c.getJSON(ctx, path, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// AllocatePort 自动分配一个端口。
func (c *Client) AllocatePort(ctx context.Context, protocol string) (int, error) {
	var res PortAllocateResult
	if err := c.decode(ctx, http.MethodPost, "/api/ports/allocate", map[string]string{"protocol": protocol}, &res); err != nil {
		return 0, err
	}
	return res.Port, nil
}

// ReleasePort 释放端口。
func (c *Client) ReleasePort(ctx context.Context, protocol string, port int) error {
	return c.decode(ctx, http.MethodPost, "/api/ports/release",
		map[string]any{"protocol": protocol, "port": port}, nil)
}
