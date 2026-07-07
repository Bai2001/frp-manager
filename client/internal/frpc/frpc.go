// Package frpc 内嵌 frpc 服务并管理其生命周期。
//
// Manager（定义在 manager.go）管理内嵌的 client.Service 生命周期：
// Start/Restart 接收 v1 配置对象而非 cfgText 字符串；生成 TOML 文本的功能
// 移到包级 MarshalConfig 函数。
package frpc

import (
	"context"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

// Start/Restart 接收 v1 配置对象而非 cfgText 字符串；生成 TOML 文本的功能移到包级 MarshalConfig 函数。
func Start(ctx context.Context, serverID string, common *v1.ClientCommonConfig, proxies []v1.ProxyConfigurer) error {
	return nil
}

func Stop(ctx context.Context, serverID string) error {
	return nil
}

func Restart(ctx context.Context, serverID string, common *v1.ClientCommonConfig, proxies []v1.ProxyConfigurer) error {
	return nil
}

func IsRunning(serverID string) bool {
	return true
}

