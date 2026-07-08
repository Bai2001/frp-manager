package main

import (
	"embed"
	"log"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/kdc/frp-manager/client/internal/config"
	"github.com/kdc/frp-manager/client/internal/settings"
)

//go:embed all:frontend/dist
var assets embed.FS

// 默认窗口尺寸（无持久化记录或记录无效时使用）
const (
	defaultWindowWidth  = 1024
	defaultWindowHeight = 768
)

func main() {
	appStruct := NewApp()

	// 预加载设置，用于恢复窗口位置/大小/最大化状态。
	// ServiceStartup 会再次加载（同一文件），二者独立无冲突。
	cfgDir, _ := config.DefaultDir()
	persisted := settings.Settings{}
	if store := settings.NewStore(filepath.Join(cfgDir, "settings.json")); store != nil {
		if s, err := store.Load(); err == nil {
			persisted = s
		}
	}

	// 先声明 app，使 SingleInstance.OnSecondInstanceLaunch 闭包能引用到它
	// （闭包延迟执行，被调用时 app 已由 application.New 赋值完成）。
	var app *application.App
	app = application.New(application.Options{
		Name:        "FRP Manager",
		Description: "基于 frp 的本地 GUI 内网穿透管理系统",
		Services: []application.Service{
			application.NewService(appStruct),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "frp-manager-instance-lock",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				// 第二实例启动时，把已运行实例的窗口前置显示
				if w := app.Window.Current(); w != nil {
					w.Show()
					w.Focus()
				}
			},
		},
		Windows: application.WindowsOptions{
			// 关闭主窗口不退出应用，托盘"退出"菜单负责真正退出
			DisableQuitOnLastWindowClosed: true,
		},
	})

	// 回填 app 引用，供 EmitLog 等使用 app.Event.Emit
	appStruct.SetApplication(app)

	// 计算初始窗口尺寸/位置：有持久化记录则恢复，否则用默认值。
	// 尺寸过小/过大或位置在屏幕外（多显示器变更后）时回退默认值/居中。
	winW, winH := defaultWindowWidth, defaultWindowHeight
	hasValidBounds := false
	if validWindowSize(persisted.WindowWidth, persisted.WindowHeight) {
		winW = persisted.WindowWidth
		winH = persisted.WindowHeight
		// 仅在尺寸有效时才校验位置；位置无效则保持默认居中。
		// 注意：此处 app.Screen 尚未就绪，位置校验在窗口创建后由
		// windowStatePersistence 重新写回时进行。恢复阶段只做尺寸过滤。
		hasValidBounds = true
	}
	opts := application.WebviewWindowOptions{
		Title:            "FRP Manager",
		Width:            winW,
		Height:           winH,
		BackgroundColour: application.NewRGB(245, 247, 250),
		URL:              "/",
	}
	// 有有效位置记录时按指定位置创建；否则居中（默认行为）。
	if hasValidBounds && validWindowPosition(persisted.WindowX, persisted.WindowY, winW, winH, app) {
		opts.InitialPosition = application.WindowXY
		opts.X = persisted.WindowX
		opts.Y = persisted.WindowY
	}

	window := app.Window.NewWithOptions(opts)
	if persisted.WindowMaximised {
		window.Maximise()
	}
	window.Show()

	// 注册窗口状态持久化：移动/缩放/最大化事件触发后节流写回 settings。
	appStruct.SetupWindowStatePersistence(window)

	// 设置系统托盘：图标 + 右键菜单（显示/退出）+ 关闭最小化到托盘
	tray := NewTray(app, window, appStruct.CloseToTray)
	tray.Setup()
	appStruct.SetTray(tray)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
