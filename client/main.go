package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	appStruct := NewApp()

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
	})

	// 回填 app 引用，供 EmitLog 等使用 app.Event.Emit
	appStruct.SetApplication(app)

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "FRP Manager",
		Width:            1024,
		Height:           768,
		BackgroundColour: application.NewRGB(245, 247, 250),
		URL:              "/",
	})
	window.Show()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
