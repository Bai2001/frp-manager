// frp-server-agent 是 FRP Manager 的服务端控制面。
// 它提供 frps 状态检测、端口池管理、域名校验等 HTTP API，
// 客户端通过本 Agent 管理服务端，不直接 SSH 到服务器执行命令。
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kdc/frp-manager/server/internal/api"
	"github.com/kdc/frp-manager/server/internal/config"
	"github.com/kdc/frp-manager/server/internal/store"
)

func main() {
	configPath := flag.String("config", "configs/agent.toml.example", "agent.toml 配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化 SQLite（骨架阶段打开即可，store 句柄后续注入 api.Server）
	db, err := store.Open(cfg.Server.Database)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer func() { _ = db.Close() }()

	apiSrv := api.New(cfg)
	httpSrv := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           apiSrv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("frp-server-agent 监听 %s", cfg.Server.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP 服务退出: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("正在关闭 ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "关闭超时: %v\n", err)
	}
	log.Println("已退出")
}
