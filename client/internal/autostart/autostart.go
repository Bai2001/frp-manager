// Package autostart 管理客户端的开机自启配置。
// 接口统一为 Enable/Disable/IsEnabled，按平台分文件实现。
package autostart

// Enable 开启开机自启。Windows 写注册表 HKCU\...\Run，
// Linux 写 ~/.config/autostart/frp-manager.desktop，
// macOS 写 ~/Library/LaunchAgents/com.kdc.frp-manager.plist。
func Enable() error { return enable() }

// Disable 关闭开机自启，删除对应注册表项/文件。
func Disable() error { return disable() }

// IsEnabled 返回当前是否已开启开机自启。
func IsEnabled() bool { return isEnabled() }
