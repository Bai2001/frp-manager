package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return New(srv.URL, "test-token")
}

func TestHealth(t *testing.T) {
	called := false
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Path != "/api/health" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": "0.1.0"})
	})
	if err := c.Health(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("未调用 health")
	}
}

func TestCapabilities(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/capabilities" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("auth header = %q", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(Capabilities{BindPort: 7000, SupportTCP: true})
	})
	caps, err := c.Capabilities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if caps.BindPort != 7000 || !caps.SupportTCP {
		t.Errorf("got %+v", caps)
	}
}

func TestCheckPort(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("protocol") != "tcp" || r.URL.Query().Get("port") != "10001" {
			t.Errorf("query = %v", r.URL.Query())
		}
		_ = json.NewEncoder(w).Encode(PortCheckResult{Protocol: "tcp", Port: 10001, Available: true, Reason: "available"})
	})
	res, err := c.CheckPort(context.Background(), "tcp", 10001)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Available {
		t.Errorf("got %+v", res)
	}
}

func TestAllocatePort(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Protocol string `json:"protocol"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Protocol != "tcp" {
			t.Errorf("protocol = %q", req.Protocol)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"protocol": "tcp", "port": 10001})
	})
	port, err := c.AllocatePort(context.Background(), "tcp")
	if err != nil {
		t.Fatal(err)
	}
	if port != 10001 {
		t.Errorf("port = %d", port)
	}
}

func TestReleasePort(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	})
	if err := c.ReleasePort(context.Background(), "tcp", 10001); err != nil {
		t.Fatal(err)
	}
}

func TestCheckDomain(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(DomainCheckResult{Domain: "app.example.com", Available: true, Reason: "available"})
	})
	res, err := c.CheckDomain(context.Background(), "http", "app.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Available {
		t.Errorf("got %+v", res)
	}
}

func TestRegisterDomain(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	})
	if err := c.RegisterDomain(context.Background(), "http", "app.example.com", "t1"); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseDomain(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	})
	if err := c.ReleaseDomain(context.Background(), "http", "app.example.com"); err != nil {
		t.Fatal(err)
	}
}

func TestErrorOn500(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "boom"})
	})
	if err := c.Health(context.Background()); err == nil {
		t.Error("500 应返回 error")
	}
}
