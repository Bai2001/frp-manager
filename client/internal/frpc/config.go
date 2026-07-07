// Package frpc 内嵌 frpc 服务的配置构造与序列化辅助。
//
// 这里不再自写 frpc.toml 的结构（Config/Auth/Proxy/Generate 已删除）,
// 改用 frp 官方 v1.* 类型直接构造配置对象，传给内嵌 client.Service;
// MarshalConfig 仅用于"查看配置"功能，把 v1 对象序列化为 frpc.toml 文本。
package frpc

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/pelletier/go-toml/v2"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

// BuildClientConfig 根据 server 信息构造 frpc 客户端配置对象。
// Auth.Method 使用 token 方式，Token 直接写入。
func BuildClientConfig(serverAddr string, serverPort int, token string) *v1.ClientCommonConfig {
	cfg := &v1.ClientCommonConfig{
		ServerAddr: serverAddr,
		ServerPort: serverPort,
	}
	cfg.Auth.Method = v1.AuthMethodToken
	cfg.Auth.Token = token
	return cfg
}

// BuildProxy 根据 tunnel 信息构造一个 v1.ProxyConfigurer。
// protocol 支持 tcp/udp/http/https；remotePort 仅 tcp/udp 使用；
// customDomain/subdomain 仅 http/https 使用。
func BuildProxy(name, protocol, localIP string, localPort int, remotePort int, customDomain, subdomain string) (v1.ProxyConfigurer, error) {
	base := v1.ProxyBaseConfig{
		Name: name,
		Type: protocol,
		ProxyBackend: v1.ProxyBackend{
			LocalIP:   localIP,
			LocalPort: localPort,
		},
	}
	switch protocol {
	case "tcp":
		return &v1.TCPProxyConfig{ProxyBaseConfig: base, RemotePort: remotePort}, nil
	case "udp":
		return &v1.UDPProxyConfig{ProxyBaseConfig: base, RemotePort: remotePort}, nil
	case "http":
		c := &v1.HTTPProxyConfig{ProxyBaseConfig: base}
		if customDomain != "" {
			c.CustomDomains = []string{customDomain}
		}
		if subdomain != "" {
			c.SubDomain = subdomain
		}
		return c, nil
	case "https":
		c := &v1.HTTPSProxyConfig{ProxyBaseConfig: base}
		if customDomain != "" {
			c.CustomDomains = []string{customDomain}
		}
		if subdomain != "" {
			c.SubDomain = subdomain
		}
		return c, nil
	default:
		return nil, fmt.Errorf("不支持的协议 %s", protocol)
	}
}

// MarshalConfig 把 v1 配置序列化为 frpc.toml 文本（用于"查看配置"功能）。
// v1 类型的 json tag 与 toml 字段名一致（都是 camelCase），通过 json 中转再 toml 编码。
func MarshalConfig(common *v1.ClientCommonConfig, proxies []v1.ProxyConfigurer) (string, error) {
	out := map[string]any{
		"serverAddr": common.ServerAddr,
		"serverPort": common.ServerPort,
		"auth":       map[string]string{"method": string(common.Auth.Method), "token": common.Auth.Token},
	}
	if len(proxies) > 0 {
		arr := make([]map[string]any, 0, len(proxies))
		for _, p := range proxies {
			arr = append(arr, proxyConfigurerToMap(p))
		}
		out["proxies"] = arr
	}
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(out); err != nil {
		return "", fmt.Errorf("序列化 frpc.toml: %w", err)
	}
	return buf.String(), nil
}

// proxyConfigurerToMap 把 ProxyConfigurer 转成 map，便于 toml 编码。
// v1 类型的 json tag 是 camelCase，与 toml 字段名一致，故用 json 中转。
// 使用 json.Number 保留数字精度，避免 int 经 interface{} 被转成 float64
// 导致 toml 输出 "3389.0" 这样的带小数点数字。
func proxyConfigurerToMap(p v1.ProxyConfigurer) map[string]any {
	b, _ := json.Marshal(p)
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var m map[string]any
	_ = dec.Decode(&m)
	return m
}
