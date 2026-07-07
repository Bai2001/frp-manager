package frpc

import (
	"strings"
	"testing"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func TestBuildClientConfig(t *testing.T) {
	cfg := BuildClientConfig("1.2.3.4", 7000, "tok")
	if cfg.ServerAddr != "1.2.3.4" || cfg.ServerPort != 7000 || cfg.Auth.Token != "tok" {
		t.Errorf("got %+v", cfg)
	}
	if cfg.Auth.Method != v1.AuthMethodToken {
		t.Errorf("Auth.Method = %q, want %q", cfg.Auth.Method, v1.AuthMethodToken)
	}
}

func TestBuildProxy_TCP(t *testing.T) {
	p, err := BuildProxy("rdp", "tcp", "127.0.0.1", 3389, 20389, "", "")
	if err != nil {
		t.Fatal(err)
	}
	tcp, ok := p.(*v1.TCPProxyConfig)
	if !ok {
		t.Fatalf("type = %T, want *TCPProxyConfig", p)
	}
	if tcp.Name != "rdp" || tcp.LocalPort != 3389 || tcp.RemotePort != 20389 {
		t.Errorf("got %+v", tcp)
	}
	if tcp.LocalIP != "127.0.0.1" {
		t.Errorf("LocalIP = %q", tcp.LocalIP)
	}
}

func TestBuildProxy_UDP(t *testing.T) {
	p, err := BuildProxy("wg", "udp", "127.0.0.1", 51820, 25180, "", "")
	if err != nil {
		t.Fatal(err)
	}
	udp, ok := p.(*v1.UDPProxyConfig)
	if !ok {
		t.Fatalf("type = %T, want *UDPProxyConfig", p)
	}
	if udp.RemotePort != 25180 {
		t.Errorf("RemotePort = %d", udp.RemotePort)
	}
}

func TestBuildProxy_HTTP_CustomDomain(t *testing.T) {
	p, err := BuildProxy("web", "http", "127.0.0.1", 3000, 0, "app.example.com", "")
	if err != nil {
		t.Fatal(err)
	}
	hp, ok := p.(*v1.HTTPProxyConfig)
	if !ok {
		t.Fatalf("type = %T, want *HTTPProxyConfig", p)
	}
	if len(hp.CustomDomains) != 1 || hp.CustomDomains[0] != "app.example.com" {
		t.Errorf("CustomDomains = %+v", hp.CustomDomains)
	}
}

func TestBuildProxy_HTTPS_Subdomain(t *testing.T) {
	p, err := BuildProxy("demo", "https", "127.0.0.1", 8443, 0, "", "demo")
	if err != nil {
		t.Fatal(err)
	}
	hp, ok := p.(*v1.HTTPSProxyConfig)
	if !ok {
		t.Fatalf("type = %T, want *HTTPSProxyConfig", p)
	}
	if hp.SubDomain != "demo" {
		t.Errorf("SubDomain = %q", hp.SubDomain)
	}
}

func TestBuildProxy_Unsupported(t *testing.T) {
	if _, err := BuildProxy("x", "stcp", "127.0.0.1", 22, 0, "", ""); err == nil {
		t.Fatal("不支持的协议应报错")
	}
}

func TestMarshalConfig(t *testing.T) {
	common := BuildClientConfig("1.2.3.4", 7000, "tok")
	p, _ := BuildProxy("rdp", "tcp", "127.0.0.1", 3389, 20389, "", "")
	out, err := MarshalConfig(common, []v1.ProxyConfigurer{p})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "1.2.3.4") {
		t.Errorf("缺少 serverAddr: %s", out)
	}
	if !strings.Contains(out, "20389") {
		t.Errorf("缺少 remotePort: %s", out)
	}
}

func TestMarshalConfig_EmptyProxies(t *testing.T) {
	common := BuildClientConfig("1.2.3.4", 7000, "tok")
	out, err := MarshalConfig(common, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "[[proxies]]") {
		t.Errorf("无 proxy 不应输出 [[proxies]]: %s", out)
	}
}
