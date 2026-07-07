package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	a := &App{}
	dir := t.TempDir()
	if err := a.InitForTest(filepath.Join(dir, "test.db"), filepath.Join(dir, "frpc-cfg")); err != nil {
		t.Fatalf("InitForTest: %v", err)
	}
	t.Cleanup(a.Close)
	return a
}

func TestAddAndListServers(t *testing.T) {
	a := newTestApp(t)
	id, err := a.AddServer(AddServerInput{
		Name: "prod", Host: "1.2.3.4", FrpsPort: 7000,
		FrpToken: "tok", AgentURL: "http://1.2.3.4:7400", AgentToken: "atok",
	})
	if err != nil {
		t.Fatalf("AddServer: %v", err)
	}
	if id == "" {
		t.Error("id 为空")
	}
	list, err := a.ListServers()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "prod" {
		t.Errorf("got %+v", list)
	}
}

func TestAddAndListTunnels(t *testing.T) {
	a := newTestApp(t)
	sid, _ := a.AddServer(AddServerInput{Name: "p", Host: "h", FrpsPort: 7000, FrpToken: "t", AgentURL: "u", AgentToken: "a"})
	tid, err := a.AddTunnel(AddTunnelInput{
		ServerID: sid, Name: "rdp", Protocol: "tcp",
		LocalIP: "127.0.0.1", LocalPort: 3389, RemotePort: 20389,
	})
	if err != nil {
		t.Fatalf("AddTunnel: %v", err)
	}
	if tid == "" {
		t.Error("tid 为空")
	}
	list, err := a.ListTunnels(sid)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "rdp" {
		t.Errorf("got %+v", list)
	}
}

func TestGenerateFrpcConfig(t *testing.T) {
	a := newTestApp(t)
	sid, _ := a.AddServer(AddServerInput{Name: "p", Host: "1.2.3.4", FrpsPort: 7000, FrpToken: "tok", AgentURL: "u", AgentToken: "a"})
	_, _ = a.AddTunnel(AddTunnelInput{ServerID: sid, Name: "rdp", Protocol: "tcp", LocalIP: "127.0.0.1", LocalPort: 3389, RemotePort: 20389})
	out, err := a.GenerateFrpcConfig(sid)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(out, `serverAddr = "1.2.3.4"`) {
		t.Errorf("缺少 serverAddr: %s", out)
	}
	if !strings.Contains(out, `remotePort = 20389`) {
		t.Errorf("缺少 remotePort: %s", out)
	}
}
