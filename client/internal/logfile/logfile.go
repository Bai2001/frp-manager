// Package logfile 负责客户端运行日志的持久化。
// 按天分文件（logs/YYYY-MM-DD.log），支持保留 N 天自动清理。
// retentionDays <= 0 时为 no-op，所有方法直接返回 nil（与 v0.1 内存日志行为一致）。
package logfile

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Writer 将日志追加写入按天分文件的日志目录。
type Writer struct {
	dir           string
	retentionDays int

	mu          sync.Mutex
	currentFile *os.File
	currentDate string
}

// New 创建 Writer。dir 为日志目录（其下会建 logs 子目录）。
// retentionDays <= 0 时返回的 Writer 为 no-op。
func New(dir string, retentionDays int) *Writer {
	return &Writer{dir: dir, retentionDays: retentionDays}
}

// Write 写入一条日志。level: info/warn/error；serverID 可为空。
// 按天切换文件，格式：RFC3339 [level] [serverID] message
func (w *Writer) Write(level, message, serverID string) error {
	if w == nil || w.retentionDays <= 0 {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	if w.currentFile == nil || w.currentDate != today {
		if w.currentFile != nil {
			_ = w.currentFile.Close()
		}
		logsDir := filepath.Join(w.dir, "logs")
		if err := os.MkdirAll(logsDir, 0o755); err != nil {
			return err
		}
		f, err := os.OpenFile(filepath.Join(logsDir, today+".log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		w.currentFile = f
		w.currentDate = today
	}
	ts := time.Now().Format(time.RFC3339)
	_, err := fmt.Fprintf(w.currentFile, "%s [%s] [%s] %s\n", ts, level, serverID, message)
	return err
}

// Close 关闭当前打开的日志文件。
func (w *Writer) Close() error {
	if w == nil || w.retentionDays <= 0 {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.currentFile != nil {
		err := w.currentFile.Close()
		w.currentFile = nil
		return err
	}
	return nil
}

// Cleanup 删除超过 retentionDays 的日志文件。
// cutoff = 今天 - retentionDays，早于 cutoff 的 YYYY-MM-DD.log 被删除。
func (w *Writer) Cleanup() error {
	if w == nil || w.retentionDays <= 0 {
		return nil
	}
	logsDir := filepath.Join(w.dir, "logs")
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	cutoff := time.Now().AddDate(0, 0, -w.retentionDays).Format("2006-01-02")
	for _, e := range entries {
		name := e.Name()
		// 文件名形如 2026-07-07.log；只处理 .log 结尾且前 10 字符为日期的文件
		if len(name) != 14 || name[10:] != ".log" {
			continue
		}
		fileDate := name[:10]
		if fileDate < cutoff {
			if err := os.Remove(filepath.Join(logsDir, name)); err != nil {
				return err
			}
		}
	}
	return nil
}
