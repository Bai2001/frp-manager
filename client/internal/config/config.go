// Package config 负责客户端本地配置目录管理。
// 骨架阶段只提供默认路径，后续模块实现时填充读写逻辑。
package config

import (
	"os"
	"path/filepath"
)

// DefaultDir 返回客户端配置与数据默认目录。
// 优先使用用户配置目录下的 frp-manager，回退到可执行文件旁的 data 目录。
func DefaultDir() (string, error) {
	if userDir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(userDir, "frp-manager"), nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "data"), nil
}

// DefaultDBPath 返回默认 SQLite 数据库路径。
func DefaultDBPath() (string, error) {
	dir, err := DefaultDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "client.db"), nil
}
