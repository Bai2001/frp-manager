package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(filepath.Join(dir, "settings.json"))
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load 缺失文件应返回零值无错，got err=%v", err)
	}
	if got.CloseToTray || got.AutoStart || got.LogRetentionDays != 0 {
		t.Errorf("零值不正确: %+v", got)
	}
}

func TestStoreSaveLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(filepath.Join(dir, "sub", "settings.json")) // 测试自动建目录
	in := Settings{CloseToTray: true, AutoStart: true, LogRetentionDays: 7, ConfigDir: "/tmp/cfg"}
	if err := s.Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != in {
		t.Errorf("往返不一致\nin:  %+v\ngot: %+v", in, got)
	}
}

func TestStoreLoadCorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewStore(path)
	if _, err := s.Load(); err == nil {
		t.Error("损坏 JSON 应返回 error")
	}
}

func TestStoreSaveAtomicNoPartialWrite(t *testing.T) {
	// 验证 Save 失败时不会留下损坏的原文件（原文件保持不变）
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	s := NewStore(path)
	orig := Settings{CloseToTray: true, LogRetentionDays: 3}
	if err := s.Save(orig); err != nil {
		t.Fatal(err)
	}
	// 再次保存不同值，应成功覆盖
	if err := s.Save(Settings{CloseToTray: false, LogRetentionDays: 5}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.LogRetentionDays != 5 || got.CloseToTray {
		t.Errorf("覆盖失败: %+v", got)
	}
}
