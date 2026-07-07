package db

import (
	"path/filepath"
	"testing"
	"time"
)

func newRepo(t *testing.T) *Repo {
	t.Helper()
	d, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	r, err := NewRepo(d)
	if err != nil {
		t.Fatalf("NewRepo: %v", err)
	}
	return r
}

func TestServers_CRUD(t *testing.T) {
	r := newRepo(t)
	now := time.Now().UTC()
	s := Server{
		ID: "s1", Name: "prod", Host: "1.2.3.4", FrpsPort: 7000,
		FrpToken: "tok", AgentURL: "http://1.2.3.4:7400", AgentToken: "atok",
		IsDefault: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := r.InsertServer(s); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, err := r.GetServer("s1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "prod" || !got.IsDefault {
		t.Errorf("got %+v", got)
	}
	all, err := r.ListServers()
	if err != nil || len(all) != 1 {
		t.Errorf("ListServers len = %d, err %v", len(all), err)
	}
	s.Name = "prod2"
	if err := r.UpdateServer(s); err != nil {
		t.Fatal(err)
	}
	got, _ = r.GetServer("s1")
	if got.Name != "prod2" {
		t.Errorf("name = %q", got.Name)
	}
	if err := r.DeleteServer("s1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetServer("s1"); err == nil {
		t.Error("删除后应查不到")
	}
}

func TestTunnels_CRUD(t *testing.T) {
	r := newRepo(t)
	now := time.Now().UTC()
	_ = r.InsertServer(Server{
		ID: "s1", Name: "prod", Host: "1.2.3.4", FrpsPort: 7000,
		FrpToken: "t", AgentURL: "http://x", AgentToken: "a",
		CreatedAt: now, UpdatedAt: now,
	})
	tu := Tunnel{
		ID: "t1", ServerID: "s1", Name: "rdp", Protocol: "tcp",
		LocalIP: "127.0.0.1", LocalPort: 3389, RemotePort: 20389,
		Enabled: true, Status: "stopped", CreatedAt: now, UpdatedAt: now,
	}
	if err := r.InsertTunnel(tu); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, err := r.GetTunnel("t1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Protocol != "tcp" || got.RemotePort != 20389 || !got.Enabled {
		t.Errorf("got %+v", got)
	}
	list, err := r.ListTunnelsByServer("s1")
	if err != nil || len(list) != 1 {
		t.Errorf("ListTunnelsByServer len = %d", len(list))
	}
	tu.Status = "running"
	if err := r.UpdateTunnel(tu); err != nil {
		t.Fatal(err)
	}
	got, _ = r.GetTunnel("t1")
	if got.Status != "running" {
		t.Errorf("status = %q", got.Status)
	}
	if err := r.DeleteTunnel("t1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetTunnel("t1"); err == nil {
		t.Error("删除后应查不到")
	}
}

func TestTunnels_DeleteServerCascade(t *testing.T) {
	r := newRepo(t)
	now := time.Now().UTC()
	_ = r.InsertServer(Server{ID: "s1", Name: "p", Host: "h", FrpsPort: 7000, FrpToken: "t", AgentURL: "u", AgentToken: "a", CreatedAt: now, UpdatedAt: now})
	_ = r.InsertTunnel(Tunnel{ID: "t1", ServerID: "s1", Name: "n", Protocol: "tcp", LocalIP: "127.0.0.1", LocalPort: 22, Enabled: true, Status: "stopped", CreatedAt: now, UpdatedAt: now})
	if err := r.DeleteServer("s1"); err != nil {
		t.Fatal(err)
	}
	list, _ := r.ListTunnelsByServer("s1")
	if len(list) != 0 {
		t.Errorf("删除 server 后 tunnels 应级联清空, len=%d", len(list))
	}
}
