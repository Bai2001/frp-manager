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
	"github.com/kdc/frp-manager/server/internal/domain"
	"github.com/kdc/frp-manager/server/internal/frps"
	"github.com/kdc/frp-manager/server/internal/portpool"
	"github.com/kdc/frp-manager/server/internal/store"
)

func main() {
	configPath := flag.String("config", "configs/agent.toml.example", "agent.toml 配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化 SQLite
	db, err := store.Open(cfg.Server.Database)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer func() { _ = db.Close() }()

	st, err := store.NewStore(db)
	if err != nil {
		log.Fatalf("创建 store 失败: %v", err)
	}

	// frps 管理（解析配置、检测状态、内嵌启停）
	frpsMgr := frps.NewManager(cfg.Frps.Config)
	frpCfg, err := frpsMgr.Config()
	if err != nil {
		log.Printf("警告: 解析 frps 配置失败（capabilities 将返回不完整）: %v", err)
	}

	// 自动启动内嵌 frps，供客户端 frpc 连接
	// frps 随 agent 进程生命周期运行，不提供运行时启停（frp 的 Close 不完全释放 vhost 端口）
	if err := frpsMgr.Start(context.Background()); err != nil {
		log.Printf("警告: 自动启动 frps 失败: %v", err)
	} else {
		log.Printf("frps 已自动启动，监听端口 %d", frpCfg.BindPort)
	}

	// 端口池与域名管理
	portsMgr := portpool.NewManager(st, frpCfg)
	domainMgr := domain.NewManager(st, &cfg.Domain)

	apiSrv := api.New(cfg, st, frpsMgr, portsMgr, domainMgr)
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
