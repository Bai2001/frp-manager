package frpsc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	src := `
bindPort = 7000
vhostHTTPPort = 80
vhostHTTPSPort = 443
subDomainHost = "frp.example.com"

[auth]
method = "token"
token = "your-frp-token"

[[allowPorts]]
start = 10000
end = 60000

[[allowPorts]]
single = 3001
`
	dir := t.TempDir()
	p := filepath.Join(dir, "frps.toml")
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Parse(p)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.BindPort != 7000 {
		t.Errorf("BindPort = %d, want 7000", cfg.BindPort)
	}
	if cfg.VhostHTTPPort == nil || *cfg.VhostHTTPPort != 80 {
		t.Errorf("VhostHTTPPort want ptr 80, got %+v", cfg.VhostHTTPPort)
	}
	if cfg.VhostHTTPSPort == nil || *cfg.VhostHTTPSPort != 443 {
		t.Errorf("VhostHTTPSPort want ptr 443, got %+v", cfg.VhostHTTPSPort)
	}
	if cfg.SubDomainHost != "frp.example.com" {
		t.Errorf("SubDomainHost = %q, want frp.example.com", cfg.SubDomainHost)
	}
	if cfg.Auth.Token != "your-frp-token" {
		t.Errorf("Auth.Token = %q", cfg.Auth.Token)
	}
	if len(cfg.AllowPorts) != 2 {
		t.Fatalf("AllowPorts len = %d, want 2", len(cfg.AllowPorts))
	}
	if cfg.AllowPorts[0].Start != 10000 || cfg.AllowPorts[0].End != 60000 {
		t.Errorf("AllowPorts[0] = %+v", cfg.AllowPorts[0])
	}
	if cfg.AllowPorts[1].Single == nil || *cfg.AllowPorts[1].Single != 3001 {
		t.Errorf("AllowPorts[1].Single want ptr 3001, got %+v", cfg.AllowPorts[1].Single)
	}
}

func TestParse_missingFile(t *testing.T) {
	if _, err := Parse(filepath.Join(t.TempDir(), "nope.toml")); err == nil {
		t.Fatal("want error for missing file")
	}
}

func TestIsPortAllowed(t *testing.T) {
	cfg := &Config{
		AllowPorts: []AllowPort{
			{Start: 10000, End: 20000},
			{Single: intPtr(3001)},
		},
	}
	cases := []struct {
		port int
		want bool
	}{
		{10000, true}, {15000, true}, {20000, true}, {3001, true},
		{9999, false}, {20001, false}, {3002, false},
	}
	for _, c := range cases {
		if got := cfg.IsPortAllowed(c.port); got != c.want {
			t.Errorf("IsPortAllowed(%d) = %v, want %v", c.port, got, c.want)
		}
	}
}

func intPtr(i int) *int { return &i }
