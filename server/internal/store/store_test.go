package store

import (
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	s, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s
}

func TestPortAllocations_InsertAndGet(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()
	pa := PortAllocation{
		ID: "p1", Protocol: "tcp", Port: 20389,
		TunnelID: "t1", Status: "allocated",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.InsertPortAllocation(pa); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, err := s.GetPortAllocation("tcp", 20389)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.TunnelID != "t1" || got.Status != "allocated" {
		t.Errorf("got %+v", got)
	}
}

func TestPortAllocations_Duplicate(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()
	pa := PortAllocation{ID: "p1", Protocol: "tcp", Port: 20389, Status: "allocated", CreatedAt: now, UpdatedAt: now}
	if err := s.InsertPortAllocation(pa); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertPortAllocation(pa); err == nil {
		t.Fatal("want duplicate error, got nil")
	}
}

func TestPortAllocations_UpdateStatus(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()
	pa := PortAllocation{ID: "p1", Protocol: "tcp", Port: 20389, Status: "allocated", CreatedAt: now, UpdatedAt: now}
	_ = s.InsertPortAllocation(pa)
	if err := s.UpdatePortAllocationStatus("p1", "released", now); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := s.GetPortAllocation("tcp", 20389)
	if got.Status != "released" {
		t.Errorf("status = %q, want released", got.Status)
	}
}

func TestDomainAllocations_InsertAndGet(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()
	da := DomainAllocation{
		ID: "d1", Protocol: "http", Domain: "app.example.com",
		TunnelID: "t1", Status: "allocated",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.InsertDomainAllocation(da); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, err := s.GetDomainAllocation("http", "app.example.com")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.TunnelID != "t1" {
		t.Errorf("got %+v", got)
	}
}

func TestDomainAllocations_UpdateStatus(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()
	da := DomainAllocation{ID: "d1", Protocol: "http", Domain: "app.example.com", Status: "allocated", CreatedAt: now, UpdatedAt: now}
	_ = s.InsertDomainAllocation(da)
	if err := s.UpdateDomainAllocationStatus("d1", "released", now); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := s.GetDomainAllocation("http", "app.example.com")
	if got.Status != "released" {
		t.Errorf("status = %q", got.Status)
	}
}
