package portpool

import (
	"path/filepath"
	"testing"

	"github.com/kdc/frp-manager/server/internal/frpsc"
	"github.com/kdc/frp-manager/server/internal/store"
)

func newManager(t *testing.T) *Manager {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	s, err := store.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	frpCfg := &frpsc.Config{
		BindPort:   7000,
		AllowPorts: []frpsc.AllowPort{{Start: 20000, End: 20100}},
	}
	return NewManager(s, frpCfg)
}

func TestCheck_Available(t *testing.T) {
	m := newManager(t)
	res, err := m.Check(nil, TCP, 20001)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Available {
		t.Errorf("Available = false, want true; reason=%s", res.Reason)
	}
}

func TestCheck_OutOfRange(t *testing.T) {
	m := newManager(t)
	res, _ := m.Check(nil, TCP, 9999)
	if res.Available {
		t.Errorf("9999 应不在 allowPorts 范围内")
	}
	if res.Reason != "out_of_allow_ports" {
		t.Errorf("reason = %q, want out_of_allow_ports", res.Reason)
	}
}

func TestCheck_AlreadyAllocated(t *testing.T) {
	m := newManager(t)
	_, err := m.Allocate(nil, TCP) // 占一个
	if err != nil {
		t.Fatal(err)
	}
	// 取第一个被分配的端口
	all, _ := m.ListAllocated(TCP)
	port := all[0]
	res, _ := m.Check(nil, TCP, port)
	if res.Available {
		t.Errorf("已分配端口 %d 应不可用", port)
	}
	if res.Reason != "already_allocated" {
		t.Errorf("reason = %q, want already_allocated", res.Reason)
	}
}

func TestAllocate(t *testing.T) {
	m := newManager(t)
	port, err := m.Allocate(nil, TCP)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if port < 20000 || port > 20100 {
		t.Errorf("port = %d, 不在允许范围", port)
	}
}

func TestAllocate_Exhausted(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	s, _ := store.NewStore(db)
	frpCfg := &frpsc.Config{AllowPorts: []frpsc.AllowPort{{Start: 20000, End: 20000}}}
	m := NewManager(s, frpCfg)
	_, err = m.Allocate(nil, TCP)
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.Allocate(nil, TCP)
	if err == nil {
		t.Fatal("端口耗尽应返回错误")
	}
}

func TestRelease(t *testing.T) {
	m := newManager(t)
	port, _ := m.Allocate(nil, TCP)
	if err := m.Release(nil, TCP, port); err != nil {
		t.Fatalf("Release: %v", err)
	}
	// 释放后应可再次分配到同一端口（唯一活跃记录已 released）
	port2, err := m.Allocate(nil, TCP)
	if err != nil {
		t.Fatal(err)
	}
	if port2 != port {
		t.Errorf("释放后未复用 %d, got %d", port, port2)
	}
}
