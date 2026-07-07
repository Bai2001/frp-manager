// Package agent 是调用 server-agent HTTP API 的客户端。
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Capabilities 对应 GET /api/capabilities 响应。
type Capabilities struct {
	FrpsRunning        bool             `json:"frps_running"`
	FrpsVersion        string           `json:"frps_version"`
	BindPort           int              `json:"bind_port"`
	AllowPorts         []AllowPortRange `json:"allow_ports"`
	SupportTCP         bool             `json:"support_tcp"`
	SupportUDP         bool             `json:"support_udp"`
	SupportHTTP        bool             `json:"support_http"`
	SupportHTTPS       bool             `json:"support_https"`
	VhostHTTPPort      int              `json:"vhost_http_port"`
	VhostHTTPSPort     int              `json:"vhost_https_port"`
	SubdomainHost      string           `json:"subdomain_host"`
	AllowedRootDomains []string         `json:"allowed_root_domains"`
}

// AllowPortRange 是 capabilities 响应里的端口范围项。
type AllowPortRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// Client 调用 server-agent API。
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// New 创建 agent 客户端。
func New(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Health 检查 agent 是否可达（GET /api/health，无需 token）。
func (c *Client) Health(ctx context.Context) error {
	_, err := c.do(ctx, http.MethodGet, "/api/health", nil, false, false)
	return err
}

// Capabilities 查询服务端能力。
func (c *Client) Capabilities(ctx context.Context) (*Capabilities, error) {
	var caps Capabilities
	if err := c.getJSON(ctx, "/api/capabilities", &caps); err != nil {
		return nil, err
	}
	return &caps, nil
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	return c.decode(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) decode(ctx context.Context, method, path string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(buf)
	}
	resp, err := c.do(ctx, method, path, bodyReader, body != nil, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("解析响应: %w", err)
		}
	}
	return nil
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader, hasBody, withAuth bool) (*http.Response, error) {
	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, err
	}
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	if withAuth {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 %s: %w", path, err)
	}
	if resp.StatusCode >= 400 {
		var errBody struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		resp.Body.Close()
		msg := errBody.Error
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("agent %d: %s", resp.StatusCode, msg)
	}
	return resp, nil
}

func (c *Client) buildURL(path string, params map[string]string) string {
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	return path + "?" + q.Encode()
}

var _ = strconv.Itoa
