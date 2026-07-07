package logfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriterNoOpWhenRetentionZero(t *testing.T) {
	w := New(t.TempDir(), 0)
	if err := w.Write("info", "msg", "srv"); err != nil {
		t.Errorf("no-op Write 应返回 nil，got %v", err)
	}
	if err := w.Cleanup(); err != nil {
		t.Errorf("no-op Cleanup 应返回 nil，got %v", err)
	}
	if err := w.Close(); err != nil {
		t.Errorf("no-op Close 应返回 nil，got %v", err)
	}
	// 确认没建 logs 目录
	if _, err := os.Stat(filepath.Join(t.TempDir(), "logs")); err == nil {
		t.Error("retentionDays=0 不应创建 logs 目录")
	}
}

func TestWriterCreatesLogByDay(t *testing.T) {
	dir := t.TempDir()
	w := New(dir, 7)
	defer w.Close()

	if err := w.Write("info", "hello", "srv1"); err != nil {
		t.Fatal(err)
	}
	if err := w.Write("error", "boom", ""); err != nil {
		t.Fatal(err)
	}

	today := time.Now().Format("2006-01-02")
	content, err := os.ReadFile(filepath.Join(dir, "logs", today+".log"))
	if err != nil {
		t.Fatalf("读取日志文件: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "[info] [srv1] hello") {
		t.Errorf("日志内容缺少 info 行: %q", s)
	}
	if !strings.Contains(s, "[error] [] boom") {
		t.Errorf("日志内容缺少 error 行: %q", s)
	}
}

func TestWriterCleanupRemovesOldFiles(t *testing.T) {
	dir := t.TempDir()
	logsDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// 造一个 10 天前的日志文件 + 今天的日志文件
	oldDate := time.Now().AddDate(0, 0, -10).Format("2006-01-02")
	today := time.Now().Format("2006-01-02")
	if err := os.WriteFile(filepath.Join(logsDir, oldDate+".log"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logsDir, today+".log"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	w := New(dir, 7) // 保留 7 天，10 天前的应被删
	if err := w.Cleanup(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(logsDir, oldDate+".log")); !os.IsNotExist(err) {
		t.Error("10 天前的日志文件应被删除")
	}
	if _, err := os.Stat(filepath.Join(logsDir, today+".log")); err != nil {
		t.Error("今天的日志文件应保留")
	}
}
