package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdc/frp-manager/server/internal/config"
	"github.com/kdc/frp-manager/server/internal/domain"
	"github.com/kdc/frp-manager/server/internal/portpool"
)

// writeAgentConfig 在临时目录生成 agent.toml 与配套 frps.toml，返回 agent.toml 路径。
// Windows 下路径含反斜杠，TOML 双引号字符串需转义；这里统一用单引号字面串避免转义问题。
func writeAgentConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	agentPath := filepath.Join(dir, "agent.toml")
	dbPath := filepath.Join(dir, "test.db")
	frpsPath := filepath.Join(dir, "frps.toml")
	body := `
[server]
addr = "127.0.0.1:0"
token = "test-token"
database = '` + dbPath + `'

[frps]
config = '` + frpsPath + `'

[domain]
allow_custom_domain = true
allowed_root_domains = ["example.com"]
subdomain_host = "frp.example.com"
`
	if err := os.WriteFile(agentPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(frpsPath, []byte(`bindPort = 7000
vhostHTTPPort = 80
vhostHTTPSPort = 443
subDomainHost = "frp.example.com"

[[allowPorts]]
start = 10000
end = 60000
`), 0o644); err != nil {
		t.Fatal(err)
	}
	return agentPath
}

func TestCapabilities(t *testing.T) {
	cfgPath := writeAgentConfig(t)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	srv := NewTestServer(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/capabilities", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp CapabilitiesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.BindPort != 7000 {
		t.Errorf("BindPort = %d, want 7000", resp.BindPort)
	}
	if !resp.SupportHTTP || resp.VhostHTTPPort != 80 {
		t.Errorf("HTTP 能力: %+v", resp)
	}
	if !resp.SupportHTTPS || resp.VhostHTTPSPort != 443 {
		t.Errorf("HTTPS 能力: %+v", resp)
	}
	if !resp.SupportTCP || !resp.SupportUDP {
		t.Errorf("TCP/UDP 应支持")
	}
	if resp.SubdomainHost != "frp.example.com" {
		t.Errorf("SubdomainHost = %q", resp.SubdomainHost)
	}
	if len(resp.AllowedRootDomains) != 1 || resp.AllowedRootDomains[0] != "example.com" {
		t.Errorf("AllowedRootDomains = %+v", resp.AllowedRootDomains)
	}
}

func TestCapabilities_Unauthorized(t *testing.T) {
	cfgPath := writeAgentConfig(t)
	cfg, _ := config.Load(cfgPath)
	srv := NewTestServer(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/capabilities", nil)
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestPortsCheck(t *testing.T) {
	cfgPath := writeAgentConfig(t)
	cfg, _ := config.Load(cfgPath)
	srv := NewTestServer(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/ports/check?protocol=tcp&port=10001", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp portpool.CheckResult
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Protocol != portpool.TCP || resp.Port != 10001 {
		t.Errorf("got %+v", resp)
	}
}

func TestPortsAllocateAndRelease(t *testing.T) {
	cfgPath := writeAgentConfig(t)
	cfg, _ := config.Load(cfgPath)
	srv := NewTestServer(t, cfg)

	// allocate
	req := httptest.NewRequest(http.MethodPost, "/api/ports/allocate", strings.NewReader(`{"protocol":"tcp"}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("allocate status=%d body=%s", rec.Code, rec.Body.String())
	}
	var alloc struct {
		Protocol string `json:"protocol"`
		Port     int    `json:"port"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &alloc)
	if alloc.Port < 10000 || alloc.Port > 60000 {
		t.Errorf("allocated port = %d, 不在范围", alloc.Port)
	}

	// release
	body := fmt.Sprintf(`{"protocol":"tcp","port":%d}`, alloc.Port)
	req2 := httptest.NewRequest(http.MethodPost, "/api/ports/release", strings.NewReader(body))
	req2.Header.Set("Authorization", "Bearer test-token")
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("release status=%d body=%s", rec2.Code, rec2.Body.String())
	}
}

func TestDomainsCheck(t *testing.T) {
	cfgPath := writeAgentConfig(t)
	cfg, _ := config.Load(cfgPath)
	srv := NewTestServer(t, cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/domains/check", strings.NewReader(`{"protocol":"http","domain":"app.example.com"}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp domain.CheckResult
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if !resp.Available {
		t.Errorf("app.example.com 应可用, got %+v", resp)
	}
}

func TestDomainsRegisterAndRelease(t *testing.T) {
	cfgPath := writeAgentConfig(t)
	cfg, _ := config.Load(cfgPath)
	srv := NewTestServer(t, cfg)

	// register
	req := httptest.NewRequest(http.MethodPost, "/api/domains/register", strings.NewReader(`{"protocol":"http","domain":"app.example.com","tunnel_id":"t1"}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("register status=%d body=%s", rec.Code, rec.Body.String())
	}

	// 再 check 应不可用
	req2 := httptest.NewRequest(http.MethodPost, "/api/domains/check", strings.NewReader(`{"protocol":"http","domain":"app.example.com"}`))
	req2.Header.Set("Authorization", "Bearer test-token")
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec2, req2)
	var resp domain.CheckResult
	_ = json.Unmarshal(rec2.Body.Bytes(), &resp)
	if resp.Available {
		t.Errorf("注册后应不可用")
	}

	// release
	req3 := httptest.NewRequest(http.MethodPost, "/api/domains/release", strings.NewReader(`{"protocol":"http","domain":"app.example.com"}`))
	req3.Header.Set("Authorization", "Bearer test-token")
	req3.Header.Set("Content-Type", "application/json")
	rec3 := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("release status=%d body=%s", rec3.Code, rec3.Body.String())
	}
}
