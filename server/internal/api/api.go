// Package api 提供 server-agent 的 HTTP 路由与 handler。
package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/kdc/frp-manager/server/internal/config"
)

// Server 持有所有 handler 依赖。
type Server struct {
	cfg *config.Config
	// TODO: store / frps / portpool / domain 依赖在后续模块实现时注入
}

// New 创建 API Server。
func New(cfg *config.Config) *Server {
	return &Server{cfg: cfg}
}

// Router 构建并返回 HTTP 路由树。
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(s.authMiddleware)

	r.Get("/api/health", s.health)
	// TODO: 以下端点在后续模块实现
	// r.Get("/api/capabilities", s.capabilities)
	// r.Get("/api/ports/check", s.checkPort)
	// r.Post("/api/ports/allocate", s.allocatePort)
	// r.Post("/api/ports/release", s.releasePort)
	// r.Post("/api/domains/check", s.checkDomain)
	// r.Post("/api/domains/register", s.registerDomain)
	// r.Post("/api/domains/release", s.releaseDomain)

	return r
}

// authMiddleware 校验 Bearer token，/api/health 放行。
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}
		got := r.Header.Get("Authorization")
		want := "Bearer " + s.cfg.Server.Token
		if got != want {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// health 健康检查端点。
func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": "0.1.0",
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
