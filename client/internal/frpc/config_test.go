package frpc

import (
	"strings"
	"testing"
)

func TestGenerate_TCPPort(t *testing.T) {
	cfg := &Config{
		ServerAddr: "1.2.3.4",
		ServerPort: 7000,
		Auth: Auth{Token: "tok"},
		Proxies: []Proxy{{
			Name: "rdp", Type: "tcp",
			LocalIP: "127.0.0.1", LocalPort: 3389, RemotePort: 20389,
		}},
	}
	out, err := Generate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `serverAddr = "1.2.3.4"`) {
		t.Errorf("缺少 serverAddr: %s", out)
	}
	if !strings.Contains(out, `serverPort = 7000`) {
		t.Errorf("缺少 serverPort: %s", out)
	}
	if !strings.Contains(out, `[auth]`) {
		t.Errorf("缺少 [auth] 表: %s", out)
	}
	if !strings.Contains(out, `token = "tok"`) {
		t.Errorf("缺少 auth.token: %s", out)
	}
	if !strings.Contains(out, `[[proxies]]`) {
		t.Errorf("缺少 [[proxies]]: %s", out)
	}
	if !strings.Contains(out, `remotePort = 20389`) {
		t.Errorf("缺少 remotePort: %s", out)
	}
}

func TestGenerate_HTTPDomain(t *testing.T) {
	cfg := &Config{
		ServerAddr: "1.2.3.4", ServerPort: 7000,
		Proxies: []Proxy{{
			Name: "web", Type: "http",
			LocalIP: "127.0.0.1", LocalPort: 3000,
			CustomDomains: []string{"app.example.com"},
		}},
	}
	out, err := Generate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `type = "http"`) {
		t.Errorf("缺少 type http: %s", out)
	}
	if !strings.Contains(out, `customDomains = ["app.example.com"]`) {
		t.Errorf("缺少 customDomains: %s", out)
	}
}

func TestGenerate_HTTPSSubdomain(t *testing.T) {
	cfg := &Config{
		ServerAddr: "1.2.3.4", ServerPort: 7000,
		Proxies: []Proxy{{
			Name: "demo", Type: "https",
			LocalIP: "127.0.0.1", LocalPort: 8443,
			Subdomain: "demo",
		}},
	}
	out, err := Generate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `subdomain = "demo"`) {
		t.Errorf("缺少 subdomain: %s", out)
	}
}

func TestGenerate_UDP(t *testing.T) {
	cfg := &Config{
		ServerAddr: "1.2.3.4", ServerPort: 7000,
		Proxies: []Proxy{{
			Name: "wg", Type: "udp",
			LocalIP: "127.0.0.1", LocalPort: 51820, RemotePort: 25180,
		}},
	}
	out, err := Generate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `type = "udp"`) || !strings.Contains(out, `remotePort = 25180`) {
		t.Errorf("缺少 udp 配置: %s", out)
	}
}

func TestGenerate_EmptyProxies(t *testing.T) {
	cfg := &Config{ServerAddr: "1.2.3.4", ServerPort: 7000}
	out, err := Generate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "[[proxies]]") {
		t.Errorf("无 proxy 不应输出 [[proxies]]: %s", out)
	}
}
