//go:build linux

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

const desktopEntry = `[Desktop Entry]
Type=Application
Name=FRP Manager
Comment=基于 frp 的本地 GUI 内网穿透管理系统
Exec=%s
Terminal=false
X-GNOME-Autostart-enabled=true
`

func desktopPath() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "autostart", "frp-manager.desktop"), nil
}

func enable() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	p, err := desktopPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(fmt.Sprintf(desktopEntry, exe)), 0o644)
}

func disable() error {
	p, err := desktopPath()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func isEnabled() bool {
	p, err := desktopPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}
