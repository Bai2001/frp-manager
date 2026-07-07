package main

import (
	_ "embed"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed build/appicon.png
var trayIcon []byte

// Tray 封装系统托盘与主窗口的生命周期绑定。
// 关闭窗口时根据 settings.CloseToTray 决定是隐藏到托盘还是真正退出。
type Tray struct {
	app     *application.App
	window  *application.WebviewWindow
	systray *application.SystemTray

	// quitting 为 true 时，WindowClosing hook 不再隐藏窗口，允许应用退出。
	// 退出菜单 / app.Quit() 前置为 true。
	quitting atomic.Bool

	// closeToTrayFn 返回当前是否启用"关闭最小化到托盘"。
	// 由 App 注入，读取实时 settings。
	closeToTrayFn func() bool
}

// NewTray 创建托盘管理器。closeToTrayFn 在每次关闭窗口时被调用以读取最新设置。
func NewTray(app *application.App, window *application.WebviewWindow, closeToTrayFn func() bool) *Tray {
	return &Tray{app: app, window: window, closeToTrayFn: closeToTrayFn}
}

// Setup 创建系统托盘图标、右键菜单，并注册窗口关闭 hook。
func (t *Tray) Setup() {
	t.systray = t.app.SystemTray.New()
	if len(trayIcon) > 0 {
		t.systray.SetIcon(trayIcon)
	}
	t.systray.SetTooltip("FRP Manager - 内网穿透管理")

	menu := t.app.NewMenu()
	menu.Add("显示主窗口").OnClick(func(ctx *application.Context) {
		t.ShowWindow()
	})
	menu.AddSeparator()
	menu.Add("退出").OnClick(func(ctx *application.Context) {
		t.Quit()
	})
	t.systray.SetMenu(menu)

	// 左键点击托盘图标显示主窗口（覆盖默认 clickHandler，避免默认行为）
	t.systray.OnClick(func() {
		t.ShowWindow()
	})
	// 显式设置右键 handler 调 OpenMenu（走托盘 bounds 定位），
	// 覆盖 applySmartDefaults 的默认 ShowMenu——两者等价，但显式设置
	// 可避免某些时序下默认 handler 被提前触发。
	t.systray.OnRightClick(func() {
		t.systray.OpenMenu()
	})

	// 拦截窗口关闭：根据设置隐藏到托盘或允许退出
	t.window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if t.quitting.Load() {
			return // 真正退出，不拦截
		}
		if t.closeToTrayFn != nil && t.closeToTrayFn() {
			t.window.Hide()
			e.Cancel()
		}
	})
}

// ShowWindow 显示并聚焦主窗口。
func (t *Tray) ShowWindow() {
	t.window.Show()
	t.window.Focus()
}

// Quit 标记退出中并触发应用退出。ServiceShutdown 会停止所有 frpc。
func (t *Tray) Quit() {
	t.quitting.Store(true)
	t.app.Quit()
}

// IsQuitting 供 App.ServiceShutdown 判断退出来源（区分窗口关闭 vs 托盘退出）。
func (t *Tray) IsQuitting() bool { return t.quitting.Load() }
