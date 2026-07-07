package frpc

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// 替身进程：Windows 用 ping，Unix 用 sleep，足够长以验证 Stop 能终止。
func fakeBinary(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		return "ping"
	}
	return "sleep"
}

func fakeArgs() []string {
	if runtime.GOOS == "windows" {
		return []string{"-n", "30", "127.0.0.1"}
	}
	return []string{"30"}
}

func TestStartAndStop(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)
	m.SetBinary(fakeBinary(t), fakeArgs()...)
	t.Cleanup(func() { _ = m.Stop(context.Background(), "s1") })

	if err := m.Start(context.Background(), "s1", "dummy config"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	// 给进程一点启动时间
	time.Sleep(200 * time.Millisecond)
	if !m.IsRunning("s1") {
		t.Errorf("启动后应 running")
	}
	if err := m.Stop(context.Background(), "s1"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	if m.IsRunning("s1") {
		t.Errorf("停止后不应 running")
	}
}

func TestStopNotRunning(t *testing.T) {
	m := NewManager(t.TempDir())
	if err := m.Stop(context.Background(), "s1"); err == nil {
		t.Error("停止未运行的进程应报错")
	}
}

func TestRestart(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)
	m.SetBinary(fakeBinary(t), fakeArgs()...)
	t.Cleanup(func() { _ = m.Stop(context.Background(), "s1") })
	_ = m.Start(context.Background(), "s1", "v1")
	time.Sleep(200 * time.Millisecond)
	if err := m.Restart(context.Background(), "s1", "v2"); err != nil {
		t.Fatalf("Restart: %v", err)
	}
	if !m.IsRunning("s1") {
		t.Errorf("重启后应 running")
	}
	_ = m.Stop(context.Background(), "s1")
}

func TestConfigFileWritten(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)
	m.SetBinary(fakeBinary(t), fakeArgs()...)
	t.Cleanup(func() { _ = m.Stop(context.Background(), "s1") })
	if err := m.Start(context.Background(), "s1", "serverAddr = \"1.2.3.4\""); err != nil {
		t.Fatal(err)
	}
	_ = m.Stop(context.Background(), "s1")
	// 配置文件应写入到 dir 下
	cfgPath := filepath.Join(dir, "s1.toml")
	if _, err := readFile(cfgPath); err != nil {
		t.Errorf("配置文件未写入: %v", err)
	}
}

func readFile(p string) (string, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
