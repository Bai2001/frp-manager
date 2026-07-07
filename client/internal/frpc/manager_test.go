package frpc

import (
	"context"
	"testing"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func TestStartStop_NotConnected(t *testing.T) {
	// 指向一个不存在的 server，frpc 会 login 失败但 Service 会被创建。
	// 验证 Start 不报错 + Stop 能清理（不强求 Start 成功，因为 login 失败可能导致 Run 立即返回）。
	m := NewManager()
	common := BuildClientConfig("127.0.0.1", 1, "tok") // port 1 不可达
	p, _ := BuildProxy("test", "tcp", "127.0.0.1", 9999, 20000, "", "")
	err := m.Start(context.Background(), "s1", common, []v1.ProxyConfigurer{p})
	if err != nil {
		// login 失败可能导致 Start 返回 error，取决于 loginFailExit 设置
		// 这里允许 err 非 nil，只要不 panic
		t.Logf("Start 返回（预期 login 失败）: %v", err)
	}
	// Stop 应能清理（即使 Start 失败）
	_ = m.Stop(context.Background(), "s1")
	if m.IsRunning("s1") {
		t.Errorf("Stop 后不应 running")
	}
}

func TestStopNotRunning(t *testing.T) {
	m := NewManager()
	if err := m.Stop(context.Background(), "s1"); err == nil {
		t.Error("停止未运行的应报错")
	}
}

func TestIsRunning_False(t *testing.T) {
	m := NewManager()
	if m.IsRunning("s1") {
		t.Error("未启动应 false")
	}
}
