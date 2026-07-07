package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kdc/frp-manager/server/internal/config"
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
