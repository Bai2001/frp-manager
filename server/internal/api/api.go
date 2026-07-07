// Package api 提供 server-agent 的 HTTP 路由与 handler。
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/kdc/frp-manager/server/internal/config"
	"github.com/kdc/frp-manager/server/internal/domain"
	"github.com/kdc/frp-manager/server/internal/frps"
	"github.com/kdc/frp-manager/server/internal/portpool"
	"github.com/kdc/frp-manager/server/internal/store"
)

// Server 持有所有 handler 依赖。
type Server struct {
	cfg     *config.Config
	store   *store.Store
	frps    *frps.Manager
	ports   *portpool.Manager
	domains *domain.Manager
}

// New 创建 API Server，注入全部依赖。
func New(cfg *config.Config, st *store.Store, f *frps.Manager, p *portpool.Manager, d *domain.Manager) *Server {
	return &Server{cfg: cfg, store: st, frps: f, ports: p, domains: d}
}

// NewTestServer 供测试用：自动从 cfg 解析 frps 配置并构造依赖。
func NewTestServer(t *testing.T, cfg *config.Config) *Server {
	t.Helper()
	db, err := store.Open(cfg.Server.Database)
	if err != nil {
		t.Fatalf("Open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	st, err := store.NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	f := frps.NewManager(cfg.Frps.Config, cfg.Frps.Binary)
	frpCfg, _ := f.Config()
	p := portpool.NewManager(st, frpCfg)
	d := domain.NewManager(st, &cfg.Domain)
	return New(cfg, st, f, p, d)
}

// Router 构建并返回 HTTP 路由树。
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(s.authMiddleware)

	r.Get("/api/health", s.health)
	r.Get("/api/capabilities", s.capabilities)
	r.Get("/api/ports/check", s.checkPort)
	r.Post("/api/ports/allocate", s.allocatePort)
	r.Post("/api/ports/release", s.releasePort)
	r.Post("/api/domains/check", s.checkDomain)
	r.Post("/api/domains/register", s.registerDomain)
	r.Post("/api/domains/release", s.releaseDomain)

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

// CapabilitiesResponse 对应设计文档 13.2 节。
type CapabilitiesResponse struct {
	FrpsRunning        bool             `json:"frps_running"`
	FrpsVersion        string           `json:"frps_version"`
	BindPort           int              `json:"bind_port"`
	AllowPorts         []AllowPortRange `json:"allow_ports"`
	SupportTCP         bool             `json:"support_tcp"`
	SupportUDP         bool             `json:"support_udp"`
	SupportHTTP        bool             `json:"support_http"`
	SupportHTTPS       bool             `json:"support_https"`
	VhostHTTPPort      int              `json:"vhost_http_port"`
	VhostHTTPSPort     int              `json:"vhost_https_port"`
	SubdomainHost      string           `json:"subdomain_host"`
	AllowedRootDomains []string         `json:"allowed_root_domains"`
}

// AllowPortRange 是 capabilities 响应里的端口范围项。
type AllowPortRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

func (s *Server) capabilities(w http.ResponseWriter, _ *http.Request) {
	cfg, err := s.frps.Config()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取 frps 配置失败: " + err.Error()})
		return
	}
	st, _ := s.frps.Status(nil)
	resp := CapabilitiesResponse{
		BindPort:           cfg.BindPort,
		FrpsRunning:        st != nil && st.Running,
		SupportTCP:         true,
		SupportUDP:         true,
		SupportHTTP:        cfg.VhostHTTPPort != nil,
		SupportHTTPS:       cfg.VhostHTTPSPort != nil,
		SubdomainHost:      cfg.SubDomainHost,
		AllowedRootDomains: s.cfg.Domain.AllowedRootDomains,
	}
	if cfg.VhostHTTPPort != nil {
		resp.VhostHTTPPort = *cfg.VhostHTTPPort
	}
	if cfg.VhostHTTPSPort != nil {
		resp.VhostHTTPSPort = *cfg.VhostHTTPSPort
	}
	for _, ap := range cfg.AllowPorts {
		if ap.Start != 0 && ap.End != 0 {
			resp.AllowPorts = append(resp.AllowPorts, AllowPortRange{Start: ap.Start, End: ap.End})
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) checkPort(w http.ResponseWriter, r *http.Request) {
	protocol := r.URL.Query().Get("protocol")
	portStr := r.URL.Query().Get("port")
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效的 port"})
		return
	}
	res, err := s.ports.Check(r.Context(), portpool.Protocol(protocol), port)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) allocatePort(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Protocol string `json:"protocol"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效请求体"})
		return
	}
	port, err := s.ports.Allocate(r.Context(), portpool.Protocol(req.Protocol))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"protocol": req.Protocol, "port": port})
}

func (s *Server) releasePort(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Protocol string `json:"protocol"`
		Port     int    `json:"port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效请求体"})
		return
	}
	if err := s.ports.Release(r.Context(), portpool.Protocol(req.Protocol), req.Port); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) checkDomain(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Protocol string `json:"protocol"`
		Domain   string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效请求体"})
		return
	}
	res, err := s.domains.Check(r.Context(), domain.Protocol(req.Protocol), req.Domain)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) registerDomain(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Protocol string `json:"protocol"`
		Domain   string `json:"domain"`
		TunnelID string `json:"tunnel_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效请求体"})
		return
	}
	if err := s.domains.Register(r.Context(), domain.Protocol(req.Protocol), req.Domain, req.TunnelID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) releaseDomain(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Protocol string `json:"protocol"`
		Domain   string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效请求体"})
		return
	}
	if err := s.domains.Release(r.Context(), domain.Protocol(req.Protocol), req.Domain); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
