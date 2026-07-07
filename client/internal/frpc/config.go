package frpc

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Config 是 frpc.toml 的完整结构。
// Auth 是嵌套 [auth] 表，避免点路径 tag 导致的 toml 写入问题。
type Config struct {
	ServerAddr string  `toml:"serverAddr"`
	ServerPort int     `toml:"serverPort"`
	Auth       Auth    `toml:"auth"`
	Proxies    []Proxy `toml:"proxies"`
}

// Auth 对应 [auth] 表。
type Auth struct {
	Method string `toml:"method,omitempty"`
	Token  string `toml:"token,omitempty"`
}

// Proxy 对应 [[proxies]] 段。
type Proxy struct {
	Name          string   `toml:"name"`
	Type          string   `toml:"type"`
	LocalIP       string   `toml:"localIP"`
	LocalPort     int      `toml:"localPort"`
	RemotePort    int      `toml:"remotePort,omitempty"`
	CustomDomains []string `toml:"customDomains,omitempty"`
	Subdomain     string   `toml:"subdomain,omitempty"`
}

// Generate 把 Config 序列化为 frpc.toml 文本。
// 空的 Proxies 不会输出 [[proxies]] 段。
// go-toml/v2 默认对纯 ASCII 字符串输出单引号字面量，frpc.toml 约定用双引号，
// 这里把值里的单引号字符串统一转换为双引号基本字符串（值内不含单引号字符，转换安全）。
func Generate(cfg *Config) (string, error) {
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	enc.SetIndentSymbol("  ")
	if err := enc.Encode(cfg); err != nil {
		return "", fmt.Errorf("序列化 frpc.toml: %w", err)
	}
	return quoteAsDouble(buf.String()), nil
}

// quoteAsDouble 把 toml 输出里的单引号字面量字符串转为双引号基本字符串。
// 仅替换形如 'xxx' 的成对单引号；值内不含单引号字符，故简单替换安全。
func quoteAsDouble(s string) string {
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		lines[i] = convertLineQuotes(ln)
	}
	return strings.Join(lines, "\n")
}

// convertLineQuotes 处理单行：找到 'xxx' 形式替换为 "xxx"。
// 一行最多一个键值对，左右各一个单引号，直接替换首尾单引号。
func convertLineQuotes(ln string) string {
	trimmed := strings.TrimSpace(ln)
	// 跳过表头/数组段 [xxx] [[xxx]]
	if strings.HasPrefix(trimmed, "[") || trimmed == "" {
		return ln
	}
	// 找到等号后的值
	eq := strings.Index(ln, "=")
	if eq == -1 {
		return ln
	}
	rest := ln[eq+1:]
	// 值部分形如 ' xxx'（可能有空格），定位单引号
	first := strings.Index(rest, "'")
	if first == -1 {
		return ln
	}
	last := strings.LastIndex(rest, "'")
	if last <= first {
		return ln
	}
	// 替换为首尾双引号
	return ln[:eq+1] + rest[:first] + "\"" + rest[first+1:last] + "\"" + rest[last+1:]
}
