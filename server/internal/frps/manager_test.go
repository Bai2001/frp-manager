package frps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kdc/frp-manager/server/internal/frpsc"
)

func writeFrpsToml(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "frps.toml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestStatus_ConfigParsed(t *testing.T) {
	p := writeFrpsToml(t, `
bindPort = 7000
vhostHTTPPort = 80
vhostHTTPSPort = 443
subDomainHost = "frp.example.com"

[auth]
token = "tok"

[[allowPorts]]
start = 10000
end = 60000
`)
	m := NewManager(p, "")
	st, err := m.Status(nil)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	// 没有 frps 进程时 Running 为 false，但配置应能解析
	if st.Running {
		t.Errorf("Running = true, want false (无真实 frps 进程)")
	}
	if st.BindPort != 7000 {
		t.Errorf("BindPort = %d, want 7000", st.BindPort)
	}
}

func TestConfig(t *testing.T) {
	p := writeFrpsToml(t, "bindPort = 7000\n"+"vhostHTTPPort = 80\n")
	m := NewManager(p, "")
	cfg, err := m.Config()
	if err != nil {
		t.Fatalf("Config: %v", err)
	}
	if cfg.BindPort != 7000 {
		t.Errorf("BindPort = %d", cfg.BindPort)
	}
	if cfg.VhostHTTPPort == nil || *cfg.VhostHTTPPort != 80 {
		t.Errorf("VhostHTTPPort = %+v", cfg.VhostHTTPPort)
	}
	_ = cfg // 也用于断言类型为 *frpsc.Config
}

func TestStatus_MissingConfig(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "nope.toml"), "")
	st, err := m.Status(nil)
	if err == nil {
		t.Fatal("缺失配置应返回 error")
	}
	if st != nil {
		t.Errorf("err 时 st 应为 nil")
	}
}

// 确保返回的 Config 类型是 *frpsc.Config
var _ = func() *frpsc.Config { var m *Manager; _, _ = m.Config(); return nil }
