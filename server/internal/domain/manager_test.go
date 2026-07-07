package domain

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdc/frp-manager/server/internal/config"
	"github.com/kdc/frp-manager/server/internal/store"
)

func newManager(t *testing.T) *Manager {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	s, _ := store.NewStore(db)
	cfg := &config.DomainConfig{
		AllowCustomDomain:  true,
		AllowedRootDomains: []string{"example.com", "frp.example.com"},
		AllowSubdomain:     true,
		SubdomainHost:      "frp.example.com",
	}
	return NewManager(s, cfg)
}

func TestCheck_InvalidFormat(t *testing.T) {
	m := newManager(t)
	res, _ := m.Check(context.Background(), HTTP, "not a domain")
	if res.Available {
		t.Error("非法域名应不可用")
	}
	if res.Reason != "invalid_format" {
		t.Errorf("reason = %q, want invalid_format", res.Reason)
	}
}

func TestCheck_NotInRootDomains(t *testing.T) {
	m := newManager(t)
	res, _ := m.Check(context.Background(), HTTP, "app.other.com")
	if res.Available || res.Reason != "not_allowed_root" {
		t.Errorf("got %+v", res)
	}
}

func TestCheck_AvailableCustom(t *testing.T) {
	m := newManager(t)
	res, _ := m.Check(context.Background(), HTTP, "app.example.com")
	if !res.Available {
		t.Errorf("got %+v", res)
	}
}

func TestCheck_AlreadyAllocated(t *testing.T) {
	m := newManager(t)
	_ = m.Register(context.Background(), HTTP, "app.example.com", "t1")
	res, _ := m.Check(context.Background(), HTTP, "app.example.com")
	if res.Available || res.Reason != "already_allocated" {
		t.Errorf("got %+v", res)
	}
}

func TestRegister_AndRelease(t *testing.T) {
	m := newManager(t)
	if err := m.Register(context.Background(), HTTP, "app.example.com", "t1"); err != nil {
		t.Fatal(err)
	}
	if err := m.Release(context.Background(), HTTP, "app.example.com"); err != nil {
		t.Fatal(err)
	}
	res, _ := m.Check(context.Background(), HTTP, "app.example.com")
	if !res.Available {
		t.Errorf("释放后应可用, got %+v", res)
	}
}

func TestCheck_Subdomain(t *testing.T) {
	m := newManager(t)
	// 子域名模式只传前缀，manager 内部拼成 demo.frp.example.com
	res, _ := m.Check(context.Background(), HTTP, "demo")
	if !res.Available {
		t.Errorf("got %+v", res)
	}
}
