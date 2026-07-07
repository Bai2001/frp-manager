package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/kdc/frp-manager/client/internal/config"
	"github.com/kdc/frp-manager/client/internal/db"
	"github.com/kdc/frp-manager/client/internal/frpc"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// 初始化客户端数据层与 frpc 管理
	dbPath, err := config.DefaultDBPath()
	if err != nil {
		println("获取默认 DB 路径失败:", err.Error())
		return
	}
	database, err := db.Open(dbPath)
	if err != nil {
		println("打开客户端数据库失败:", err.Error())
		return
	}
	repo, err := db.NewRepo(database)
	if err != nil {
		println("初始化 db repo 失败:", err.Error())
		return
	}
	frpcConfigDir, _ := config.DefaultDir()
	app.Init(repo, frpc.NewManager(frpcConfigDir))
	app.SetDatabase(database)

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "FRP Manager",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 245, G: 247, B: 250, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		// 单实例锁：防止重复启动。托盘与"关闭最小化到托盘"因 Wails v2.12
		// 托盘 API 不在 options.App 直接暴露（仅内部 menumanager），留 v0.2。
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "frp-manager-instance-lock",
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
