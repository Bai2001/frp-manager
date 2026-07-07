package frps

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func writeFrpsToml(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "frps.toml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestConfig(t *testing.T) {
	p := writeFrpsToml(t, `bindPort = 7000
vhostHTTPPort = 80
vhostHTTPSPort = 443
subDomainHost = "frp.example.com"

[auth]
token = "tok"

[[allowPorts]]
start = 10000
end = 60000
`)
	m := NewManager(p)
	cfg, err := m.Config()
	if err != nil {
		t.Fatalf("Config: %v", err)
	}
	if cfg.BindPort != 7000 {
		t.Errorf("BindPort = %d", cfg.BindPort)
	}
	if cfg.VhostHTTPPort != 80 {
		t.Errorf("VhostHTTPPort = %d", cfg.VhostHTTPPort)
	}
	if cfg.SubDomainHost != "frp.example.com" {
		t.Errorf("SubDomainHost = %q", cfg.SubDomainHost)
	}
	if cfg.Auth.Token != "tok" {
		t.Errorf("Auth.Token = %q", cfg.Auth.Token)
	}
	if len(cfg.AllowPorts) != 1 || cfg.AllowPorts[0].Start != 10000 || cfg.AllowPorts[0].End != 60000 {
		t.Errorf("AllowPorts = %+v", cfg.AllowPorts)
	}
}

func TestStatus_NotRunning(t *testing.T) {
	p := writeFrpsToml(t, `bindPort = 7000`)
	m := NewManager(p)
	st, err := m.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.Running {
		t.Errorf("Running = true, want false（无真实 frps 监听 7000）")
	}
	if st.BindPort != 7000 {
		t.Errorf("BindPort = %d", st.BindPort)
	}
}

func TestConfig_MissingFile(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "nope.toml"))
	if _, err := m.Config(); err == nil {
		t.Fatal("缺失配置应返回 error")
	}
}
