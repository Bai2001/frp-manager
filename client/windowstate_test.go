package main

import "testing"

func TestValidWindowSize(t *testing.T) {
	cases := []struct {
		name        string
		w, h        int
		want        bool
		description string
	}{
		{"零值", 0, 0, false, "无记录或窗口销毁"},
		{"仅宽为零", 0, 600, false, "尺寸不完整"},
		{"仅高为零", 800, 0, false, "尺寸不完整"},
		{"刚好下限", 800, 600, true, "边界值有效"},
		{"宽不足下限", 799, 600, false, "宽度过小"},
		{"高不足下限", 800, 599, false, "高度过小"},
		{"典型尺寸", 1024, 768, true, "正常窗口"},
		{"宽度过大", 8193, 600, false, "跨多屏异常值"},
		{"高度过大", 800, 8193, false, "跨多屏异常值"},
		{"上限边界", 8192, 8192, true, "边界值有效"},
		{"负数", -100, 600, false, "异常负值"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := validWindowSize(c.w, c.h)
			if got != c.want {
				t.Fatalf("validWindowSize(%d, %d) = %v, want %v (%s)",
					c.w, c.h, got, c.want, c.description)
			}
		})
	}
}

// validWindowPosition 在 app 为 nil 时宽松放行（不依赖运行时）。
// 验证该分支不会误判有效尺寸，避免在没有 Wails 运行时的环境里阻断持久化。
func TestValidWindowPosition_NilApp(t *testing.T) {
	if !validWindowPosition(100, 100, 800, 600, nil) {
		t.Fatal("app 为 nil 时应宽松放行，返回 true")
	}
	if !validWindowPosition(-9999, -9999, 800, 600, nil) {
		t.Fatal("app 为 nil 时应宽松放行，返回 true")
	}
}
