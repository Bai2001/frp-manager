package portpool

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/kdc/frp-manager/server/internal/frpsc"
	"github.com/kdc/frp-manager/server/internal/portprobe"
	"github.com/kdc/frp-manager/server/internal/store"
)

// Manager 是端口池的真实实现。
type Manager struct {
	store  *store.Store
	frpCfg *frpsc.Config
}

// NewManager 创建端口池管理器。
// frpCfg 用于读取 allowPorts 范围；为 nil 表示不限制。
func NewManager(s *store.Store, frpCfg *frpsc.Config) *Manager {
	return &Manager{store: s, frpCfg: frpCfg}
}

// Check 检查指定端口是否可用：范围 + 已分配 + 本地监听探测。
func (m *Manager) Check(_ context.Context, p Protocol, port int) (*CheckResult, error) {
	if m.frpCfg != nil && !m.frpCfg.IsPortAllowed(port) {
		return &CheckResult{Protocol: p, Port: port, Available: false, Reason: "out_of_allow_ports"}, nil
	}
	pa, err := m.store.GetPortAllocation(string(p), port)
	if err == nil && pa.Status == "allocated" {
		return &CheckResult{Protocol: p, Port: port, Available: false, Reason: "already_allocated"}, nil
	}
	avail, err := probe(p, port)
	if err != nil {
		return nil, err
	}
	if !avail {
		return &CheckResult{Protocol: p, Port: port, Available: false, Reason: "in_use"}, nil
	}
	return &CheckResult{Protocol: p, Port: port, Available: true, Reason: "available"}, nil
}

// Allocate 在允许范围内自动寻找一个可用端口并记录占用。
func (m *Manager) Allocate(_ context.Context, p Protocol) (int, error) {
	ranges := m.allowRanges()
	for _, r := range ranges {
		for port := r.start; port <= r.end; port++ {
			pa, err := m.store.GetPortAllocation(string(p), port)
			if err == nil {
				// 已有记录：allocated 跳过；released 复用（更新回 allocated）
				if pa.Status == "allocated" {
					continue
				}
				avail, err := probe(p, port)
				if err != nil {
					return 0, err
				}
				if !avail {
					continue
				}
				if err := m.store.UpdatePortAllocationStatus(pa.ID, "allocated", time.Now().UTC()); err != nil {
					return 0, fmt.Errorf("复用端口占用记录: %w", err)
				}
				return port, nil
			}
			avail, err := probe(p, port)
			if err != nil {
				return 0, err
			}
			if !avail {
				continue
			}
			now := time.Now().UTC()
			if err := m.store.InsertPortAllocation(store.PortAllocation{
				ID: uuid.NewString(), Protocol: string(p), Port: port,
				Status: "allocated", CreatedAt: now, UpdatedAt: now,
			}); err != nil {
				return 0, fmt.Errorf("记录端口占用: %w", err)
			}
			return port, nil
		}
	}
	return 0, errors.New("无可用端口")
}

// Release 释放端口占用（标记为 released）。
func (m *Manager) Release(_ context.Context, p Protocol, port int) error {
	pa, err := m.store.GetPortAllocation(string(p), port)
	if err != nil {
		return fmt.Errorf("端口未分配: %w", err)
	}
	return m.store.UpdatePortAllocationStatus(pa.ID, "released", time.Now().UTC())
}

// ListAllocated 返回指定协议当前 allocated 的端口列表。
func (m *Manager) ListAllocated(p Protocol) ([]int, error) {
	var out []int
	for _, r := range m.allowRanges() {
		for port := r.start; port <= r.end; port++ {
			pa, err := m.store.GetPortAllocation(string(p), port)
			if err == nil && pa.Status == "allocated" {
				out = append(out, port)
			}
		}
	}
	return out, nil
}

type portRange struct{ start, end int }

func (m *Manager) allowRanges() []portRange {
	if m.frpCfg == nil || len(m.frpCfg.AllowPorts) == 0 {
		return []portRange{{0, 65535}}
	}
	var out []portRange
	for _, ap := range m.frpCfg.AllowPorts {
		if ap.Single != nil {
			out = append(out, portRange{*ap.Single, *ap.Single})
		}
		if ap.Start != 0 && ap.End != 0 {
			out = append(out, portRange{ap.Start, ap.End})
		}
	}
	return out
}

func probe(p Protocol, port int) (bool, error) {
	switch p {
	case TCP:
		return portprobe.TCPAvailable(port)
	case UDP:
		return portprobe.UDPAvailable(port)
	default:
		return false, fmt.Errorf("不支持的协议 %s", p)
	}
}
