// Package frpsc 解析 frps.toml 配置文件，提取 agent 需要的能力字段。
package frpsc

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Config 是 frps.toml 中 agent 关心的子集。
type Config struct {
	BindPort       int         `toml:"bindPort"`
	VhostHTTPPort  *int        `toml:"vhostHTTPPort"`
	VhostHTTPSPort *int        `toml:"vhostHTTPSPort"`
	SubDomainHost  string      `toml:"subDomainHost"`
	Auth           Auth        `toml:"auth"`
	AllowPorts     []AllowPort `toml:"allowPorts"`
}

// Auth 是 frps 鉴权配置。
type Auth struct {
	Method string `toml:"method"`
	Token  string `toml:"token"`
}

// AllowPort 描述一段允许的端口范围或单个端口。
type AllowPort struct {
	Start  int  `toml:"start"`
	End    int  `toml:"end"`
	Single *int `toml:"single"`
}

// Parse 读取并解析指定路径的 frps.toml。
func Parse(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 frps 配置 %s: %w", path, err)
	}
	var c Config
	if err := toml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("解析 frps 配置 %s: %w", path, err)
	}
	return &c, nil
}

// IsPortAllowed 判断 port 是否落在 allowPorts 范围内。
// 若 allowPorts 为空表示不限制（与 frps 行为一致）。
func (c *Config) IsPortAllowed(port int) bool {
	if len(c.AllowPorts) == 0 {
		return true
	}
	for _, r := range c.AllowPorts {
		if r.Single != nil && *r.Single == port {
			return true
		}
		if r.Start != 0 && r.End != 0 && port >= r.Start && port <= r.End {
			return true
		}
	}
	return false
}
