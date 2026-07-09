//go:build windows

package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/w32"
)

// windowsWindowOptions 根据 theme_mode 生成 Windows 窗口主题选项。
// 自定义标题栏颜色与前端 content-bg 对齐，避免深色内容 + 白色标题栏。
func windowsWindowOptions(themeMode string) application.WindowsWindow {
	return application.WindowsWindow{
		Theme: mapWindowsTheme(themeMode),
		CustomTheme: application.ThemeSettings{
			DarkModeActive: &application.WindowTheme{
				TitleBarColour:  w32.RGBptr(20, 22, 29),    // #14161d
				TitleTextColour: w32.RGBptr(229, 231, 235), // #e5e7eb
				BorderColour:    w32.RGBptr(31, 36, 48),    // #1f2430
			},
			DarkModeInactive: &application.WindowTheme{
				TitleBarColour:  w32.RGBptr(12, 14, 20),    // #0c0e14
				TitleTextColour: w32.RGBptr(156, 163, 175), // #9ca3af
				BorderColour:    w32.RGBptr(31, 36, 48),
			},
			LightModeActive: &application.WindowTheme{
				TitleBarColour:  w32.RGBptr(244, 245, 247), // #f4f5f7
				TitleTextColour: w32.RGBptr(31, 41, 55),    // #1f2937
				BorderColour:    w32.RGBptr(229, 231, 235), // #e5e7eb
			},
			LightModeInactive: &application.WindowTheme{
				TitleBarColour:  w32.RGBptr(248, 250, 252), // #f8fafc
				TitleTextColour: w32.RGBptr(107, 114, 128), // #6b7280
				BorderColour:    w32.RGBptr(229, 231, 235),
			},
		},
	}
}

func mapWindowsTheme(themeMode string) application.Theme {
	switch themeMode {
	case "dark":
		return application.Dark
	case "light":
		return application.Light
	default:
		return application.SystemDefault
	}
}

// resolveNativeDark 解析标题栏是否应使用深色。
func resolveNativeDark(themeMode string) bool {
	switch themeMode {
	case "dark":
		return true
	case "light":
		return false
	default:
		return w32.IsCurrentlyDarkMode()
	}
}

// backgroundColourForTheme 返回与主题匹配的窗口背景色（减少启动闪白/闪黑）。
func backgroundColourForTheme(themeMode string) application.RGBA {
	if resolveNativeDark(themeMode) {
		return application.NewRGB(20, 22, 29)
	}
	return application.NewRGB(244, 245, 247)
}

// applyNativeTheme 运行时更新原生标题栏 / 边框 / 窗口背景色。
func applyNativeTheme(win *application.WebviewWindow, themeMode string) {
	if win == nil {
		return
	}
	isDark := resolveNativeDark(themeMode)
	// 同步 WebView 背景，避免主题切换时露底
	win.SetBackgroundColour(backgroundColourForTheme(themeMode))

	native := win.NativeWindow()
	if native == nil {
		return
	}
	hwnd := uintptr(native)
	if !w32.SupportsThemes() {
		return
	}
	if w32.AllowDarkModeForWindow != nil {
		w32.AllowDarkModeForWindow(w32.HWND(hwnd), isDark)
	}
	w32.SetTheme(hwnd, isDark)

	if !w32.SupportsCustomThemes() {
		return
	}
	if isDark {
		w32.SetTitleBarColour(hwnd, w32.RGB(20, 22, 29))
		w32.SetTitleTextColour(hwnd, w32.RGB(229, 231, 235))
		w32.SetBorderColour(hwnd, w32.RGB(31, 36, 48))
	} else {
		w32.SetTitleBarColour(hwnd, w32.RGB(244, 245, 247))
		w32.SetTitleTextColour(hwnd, w32.RGB(31, 41, 55))
		w32.SetBorderColour(hwnd, w32.RGB(229, 231, 235))
	}
	// 触发非客户区重绘
	w32.InvalidateRect(w32.HWND(hwnd), nil, true)
}
