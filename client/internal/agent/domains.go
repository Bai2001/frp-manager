package agent

import (
	"context"
	"net/http"
)

// DomainCheckResult 对应 POST /api/domains/check 响应。
type DomainCheckResult struct {
	Domain    string `json:"domain"`
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
}

// CheckDomain 校验域名可用性。
func (c *Client) CheckDomain(ctx context.Context, protocol, domain string) (*DomainCheckResult, error) {
	var res DomainCheckResult
	if err := c.decode(ctx, http.MethodPost, "/api/domains/check",
		map[string]string{"protocol": protocol, "domain": domain}, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// RegisterDomain 注册域名占用。
func (c *Client) RegisterDomain(ctx context.Context, protocol, domain, tunnelID string) error {
	return c.decode(ctx, http.MethodPost, "/api/domains/register",
		map[string]string{"protocol": protocol, "domain": domain, "tunnel_id": tunnelID}, nil)
}

// ReleaseDomain 释放域名。
func (c *Client) ReleaseDomain(ctx context.Context, protocol, domain string) error {
	return c.decode(ctx, http.MethodPost, "/api/domains/release",
		map[string]string{"protocol": protocol, "domain": domain}, nil)
}
