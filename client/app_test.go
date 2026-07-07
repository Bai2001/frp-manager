package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdc/frp-manager/client/internal/settings"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	a := &App{}
	dir := t.TempDir()
	if err := a.InitForTest(filepath.Join(dir, "test.db")); err != nil {
		t.Fatalf("InitForTest: %v", err)
	}
	t.Cleanup(a.Close)
	return a
}

// newMockAgent 启动一个 mock agent HTTP 服务，处理端口/域名校验与注册。
// 返回 base URL，供测试 server 的 agent_url 使用。
func newMockAgent(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ports/check":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"protocol":  r.URL.Query().Get("protocol"),
				"port":      20389,
				"available": true,
				"reason":    "available",
			})
		case "/api/domains/check":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"domain":    "app.example.com",
				"available": true,
				"reason":    "available",
			})
		case "/api/domains/register", "/api/domains/release",
			"/api/ports/allocate", "/api/ports/release":
			_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
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
	agentURL := newMockAgent(t)
	sid, _ := a.AddServer(AddServerInput{Name: "p", Host: "h", FrpsPort: 7000, FrpToken: "t", AgentURL: agentURL, AgentToken: "a"})
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
	agentURL := newMockAgent(t)
	sid, _ := a.AddServer(AddServerInput{Name: "p", Host: "1.2.3.4", FrpsPort: 7000, FrpToken: "tok", AgentURL: agentURL, AgentToken: "a"})
	_, _ = a.AddTunnel(AddTunnelInput{ServerID: sid, Name: "rdp", Protocol: "tcp", LocalIP: "127.0.0.1", LocalPort: 3389, RemotePort: 20389})
	out, err := a.GenerateFrpcConfig(sid)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// MarshalConfig 用 go-toml/v2 编码，字符串默认单引号字面量；
	// 这里只验证关键字段存在，不强制引号风格（仅"查看配置"展示用）。
	if !strings.Contains(out, "1.2.3.4") {
		t.Errorf("缺少 serverAddr: %s", out)
	}
	if !strings.Contains(out, "20389") {
		t.Errorf("缺少 remotePort: %s", out)
	}
}

func TestSettingsSaveLoad(t *testing.T) {
	a := newTestApp(t)
	in := settings.Settings{CloseToTray: true, LogRetentionDays: 7}
	// 不测 AutoStart 真实写注册表，避免污染环境
	if err := a.SaveSettings(in); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	got, err := a.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got.CloseToTray != true || got.LogRetentionDays != 7 {
		t.Errorf("设置往返不一致: %+v", got)
	}
}

func TestExportImportDataRoundtrip(t *testing.T) {
	a := newTestApp(t)
	agentURL := newMockAgent(t)
	sid, _ := a.AddServer(AddServerInput{Name: "prod", Host: "1.2.3.4", FrpsPort: 7000, FrpToken: "tok", AgentURL: agentURL, AgentToken: "a"})
	_, _ = a.AddTunnel(AddTunnelInput{ServerID: sid, Name: "rdp", Protocol: "tcp", LocalIP: "127.0.0.1", LocalPort: 3389, RemotePort: 20389})

	// 导出
	raw, err := a.ExportData()
	if err != nil {
		t.Fatalf("ExportData: %v", err)
	}
	if !strings.Contains(raw, "prod") || !strings.Contains(raw, "rdp") {
		t.Errorf("导出数据缺少内容: %s", raw)
	}

	// 导入前清空验证：导入会全量替换，导入后应能查到
	a2 := newTestApp(t)
	if err := a2.ImportData(raw); err != nil {
		t.Fatalf("ImportData: %v", err)
	}
	servers, _ := a2.ListServers()
	if len(servers) != 1 || servers[0].Name != "prod" {
		t.Errorf("导入后服务器不匹配: %+v", servers)
	}
	tunnels, _ := a2.ListTunnels(sid)
	if len(tunnels) != 1 || tunnels[0].Name != "rdp" {
		t.Errorf("导入后映射不匹配: %+v", tunnels)
	}
}

func TestImportDataCorruptJson(t *testing.T) {
	a := newTestApp(t)
	if err := a.ImportData("{bad json"); err == nil {
		t.Error("损坏 JSON 应返回 error")
	}
}
