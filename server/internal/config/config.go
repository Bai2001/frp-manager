// Package config 加载 server-agent 的 agent.toml 配置。
package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Config 是 agent 运行时配置。
type Config struct {
	Server  ServerConfig  `toml:"server"`
	Frps    FrpsConfig    `toml:"frps"`
	Domain  DomainConfig  `toml:"domain"`
}

// ServerConfig 是 Agent HTTP API 自身的监听与鉴权配置。
type ServerConfig struct {
	Addr    string `toml:"addr"`
	Token   string `toml:"token"`
	Database string `toml:"database"`
}

// FrpsConfig 描述 frps 二进制与配置文件位置。
type FrpsConfig struct {
	Binary string `toml:"binary"`
	Config string `toml:"config"`
	LogDir string `toml:"log_dir"`
}

// DomainConfig 是 HTTP/HTTPS 域名策略。
type DomainConfig struct {
	AllowCustomDomain  bool     `toml:"allow_custom_domain"`
	AllowedRootDomains []string `toml:"allowed_root_domains"`
	AllowSubdomain     bool     `toml:"allow_subdomain"`
	SubdomainHost      string   `toml:"subdomain_host"`
}

// Load 从 path 读取并解析 agent.toml。
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置 %s: %w", path, err)
	}
	var c Config
	if err := toml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("解析配置 %s: %w", path, err)
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) validate() error {
	if c.Server.Addr == "" {
		return fmt.Errorf("server.addr 不能为空")
	}
	if c.Server.Token == "" {
		return fmt.Errorf("server.token 不能为空")
	}
	if c.Server.Database == "" {
		c.Server.Database = "data/server.db"
	}
	return nil
}
