package frps

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strconv"
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

// freePort 返回一个空闲 TCP 端口，避免与真实 frps 服务冲突。
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("获取空闲端口: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestStatus_NotRunning(t *testing.T) {
	port := freePort(t)
	p := writeFrpsToml(t, `bindPort = `+strconv.Itoa(port)+`
`)
	m := NewManager(p)
	st, err := m.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.Running {
		t.Errorf("Running = true, want false（端口 %d 无 frps 监听）", port)
	}
	if st.BindPort != port {
		t.Errorf("BindPort = %d, want %d", st.BindPort, port)
	}
}

func TestConfig_MissingFile(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "nope.toml"))
	if _, err := m.Config(); err == nil {
		t.Fatal("缺失配置应返回 error")
	}
}
