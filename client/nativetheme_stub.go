//go:build !windows

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// windowsWindowOptions 非 Windows 平台返回空选项。
func windowsWindowOptions(themeMode string) application.WindowsWindow {
	_ = themeMode
	return application.WindowsWindow{}
}

// backgroundColourForTheme 非 Windows 也按 theme_mode 给启动背景色。
func backgroundColourForTheme(themeMode string) application.RGBA {
	if themeMode == "dark" {
		return application.NewRGB(20, 22, 29)
	}
	return application.NewRGB(244, 245, 247)
}

// applyNativeTheme 非 Windows 无原生标题栏主题可调。
func applyNativeTheme(win *application.WebviewWindow, themeMode string) {
	if win == nil {
		return
	}
	win.SetBackgroundColour(backgroundColourForTheme(themeMode))
}
