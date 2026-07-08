package main

import (
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// windowStatePersistence 负责监听窗口移动/缩放/最大化事件，
// 节流写回 settings，使下次启动能恢复上次窗口位置和大小。
//
// 设计要点：
//   - 移动和缩放事件触发频率高（拖动/拉伸过程中持续触发），
//     用 500ms 节流定时器合并写盘，避免 IO 风暴。
//   - 最大化/还原/最小化状态变更立即写（低频且语义关键）。
//   - 最大化状态下不记录位置/尺寸（恢复时直接最大化即可，
//     避免把最大化时的全屏坐标当作普通位置写回）。
//   - 写回时仅更新窗口状态字段，保留其余设置字段不变。
type windowStatePersistence struct {
	app    *App
	window *application.WebviewWindow

	mu       sync.Mutex
	timer    *time.Timer
	debounce time.Duration
}

// SetupWindowStatePersistence 在 App 上注册窗口状态持久化。
// 由 main.go 在窗口创建后调用。
func (a *App) SetupWindowStatePersistence(window *application.WebviewWindow) {
	p := &windowStatePersistence{app: a, window: window, debounce: 500 * time.Millisecond}
	p.register()
}

func (p *windowStatePersistence) register() {
	// 移动和缩放：节流写回位置/尺寸
	p.window.OnWindowEvent(events.Common.WindowDidMove, func(e *application.WindowEvent) {
		p.scheduleBoundsSave()
	})
	p.window.OnWindowEvent(events.Common.WindowDidResize, func(e *application.WindowEvent) {
		p.scheduleBoundsSave()
	})
	// 最大化/还原：立即写状态（低频）
	p.window.OnWindowEvent(events.Common.WindowMaximise, func(e *application.WindowEvent) {
		p.saveMaximised(true)
	})
	p.window.OnWindowEvent(events.Common.WindowUnMaximise, func(e *application.WindowEvent) {
		p.saveMaximised(false)
		// 取消最大化后会触发 WindowDidResize，但为确保位置/尺寸及时落盘，主动记录一次
		p.scheduleBoundsSave()
	})
	p.window.OnWindowEvent(events.Common.WindowRestore, func(e *application.WindowEvent) {
		p.scheduleBoundsSave()
	})
}

// scheduleBoundsSave 节流写回窗口位置和尺寸。
// 拖动/拉伸过程中事件高频触发，合并为每 500ms 至多一次写盘。
func (p *windowStatePersistence) scheduleBoundsSave() {
	// 最大化状态下不记录位置/尺寸（恢复时直接最大化即可）
	if p.window.IsMaximised() {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.timer != nil {
		p.timer.Stop()
	}
	p.timer = time.AfterFunc(p.debounce, p.flushBounds)
}

// flushBounds 读取当前窗口位置/尺寸并写回 settings。
func (p *windowStatePersistence) flushBounds() {
	x, y := p.window.Position()
	w, h := p.window.Size()
	if w <= 0 || h <= 0 {
		return // 窗口已销毁或尺寸异常
	}
	p.app.updateWindowBounds(x, y, w, h)
}

// saveMaximised 立即写回最大化状态。
func (p *windowStatePersistence) saveMaximised(maximised bool) {
	p.app.updateWindowMaximised(maximised)
}

// updateWindowBounds 更新缓存的窗口位置/尺寸并持久化。
// 仅写窗口状态字段，保留其余设置不变。
func (a *App) updateWindowBounds(x, y, w, h int) {
	if a.settingsStore == nil {
		return
	}
	a.settings.WindowX = x
	a.settings.WindowY = y
	a.settings.WindowWidth = w
	a.settings.WindowHeight = h
	_ = a.settingsStore.Save(a.settings)
}

// updateWindowMaximised 更新缓存的窗口最大化状态并持久化。
func (a *App) updateWindowMaximised(maximised bool) {
	if a.settingsStore == nil {
		return
	}
	a.settings.WindowMaximised = maximised
	_ = a.settingsStore.Save(a.settings)
}
