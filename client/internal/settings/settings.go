// Package settings 负责客户端应用设置的持久化读写。
// 设置存放在配置目录下的 settings.json，采用原子写（临时文件 + rename）。
package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings 是客户端应用设置。
// 零值约定：LogRetentionDays=0 表示不落盘日志（与 v0.1 内存日志行为一致）。
type Settings struct {
	CloseToTray      bool   `json:"close_to_tray"`      // 关闭窗口时最小化到托盘
	AutoStart        bool   `json:"auto_start"`         // 开机自启
	LogRetentionDays int    `json:"log_retention_days"` // 日志保留天数，0=不落盘
	ConfigDir        string `json:"config_dir"`         // 配置目录（只读展示，当前不支持修改）
	// ThemeMode 外观主题：system | light | dark。
	// 空字符串或未知值由前端按 system 处理（兼容旧配置）。
	ThemeMode string `json:"theme_mode"`

	// 窗口状态持久化（DIP 坐标，与 Wails Position()/Size() 返回值一致）。
	// 由窗口移动/缩放/最大化事件自动写回，前端设置页不感知这些字段。
	// WindowWidth/WindowHeight 为 0 时表示无记录，使用默认尺寸。
	WindowMaximised bool `json:"window_maximised"`
	WindowX         int  `json:"window_x"`
	WindowY         int  `json:"window_y"`
	WindowWidth     int  `json:"window_width"`
	WindowHeight    int  `json:"window_height"`
}

// Store 负责读写 settings.json。
type Store struct {
	path string
}

// NewStore 创建 Store。path 为 settings.json 的完整路径。
func NewStore(path string) *Store {
	return &Store{path: path}
}

// Load 读取设置。文件不存在时返回零值 Settings 和 nil error（向后兼容首次运行）。
// 文件损坏（JSON 解析失败）时返回零值和 error，调用方可选择忽略并重置。
func (s *Store) Load() (Settings, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return Settings{}, nil
		}
		return Settings{}, err
	}
	var out Settings
	if err := json.Unmarshal(data, &out); err != nil {
		return Settings{}, err
	}
	return out, nil
}

// Save 写入设置。先写临时文件再 rename，保证原子性。
// 父目录不存在时自动创建。
func (s *Store) Save(in Settings) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), "settings-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // rename 成功后删除临时文件（rename 跨文件会失败，此处同目录）
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, s.path)
}
