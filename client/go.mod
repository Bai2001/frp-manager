module github.com/kdc/frp-manager/client

go 1.26

require (
	github.com/fatedier/frp v0.69.1
	github.com/google/uuid v1.6.0
	github.com/pelletier/go-toml/v2 v2.4.3
	github.com/wailsapp/wails/v3 v3.0.0-alpha2.115
	golang.org/x/sys v0.44.0
	modernc.org/sqlite v1.53.0
)

require (
	github.com/Azure/go-ntlmssp v0.1.0 // indirect
	github.com/adrg/xdg v0.5.3 // indirect
	github.com/armon/go-socks5 v0.0.0-20160902184237-e75332964ef5 // indirect
	github.com/coder/websocket v1.8.14 // indirect
	github.com/coreos/go-oidc/v3 v3.14.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatedier/golib v0.7.0 // indirect
	github.com/go-jose/go-jose/v4 v4.0.5 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/hashicorp/yamux v0.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jchv/go-winloader v0.0.0-20250406163304-c1995be93bd1 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/klauspost/reedsolomon v1.12.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/pion/dtls/v3 v3.0.10 // indirect
	github.com/pion/logging v0.2.4 // indirect
	github.com/pion/stun/v3 v3.1.1 // indirect
	github.com/pion/transport/v4 v4.0.1 // indirect
	github.com/pires/go-proxyproto v0.7.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/quic-go/quic-go v0.55.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/samber/lo v1.49.1 // indirect
	github.com/songgao/water v0.0.0-20200317203138-2b4b6d7c09d8 // indirect
	github.com/spf13/cobra v1.8.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/templexxx/cpu v0.1.1 // indirect
	github.com/templexxx/xorsimd v0.4.3 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/vishvananda/netlink v1.3.0 // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	github.com/wlynxg/anet v0.0.5 // indirect
	github.com/xtaci/kcp-go/v5 v5.6.13 // indirect
	golang.org/x/crypto v0.51.0 // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/net v0.54.0 // indirect
	golang.org/x/oauth2 v0.28.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	golang.org/x/time v0.10.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
	golang.zx2c4.com/wireguard v0.0.0-20231211153847-12269c276173 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/apimachinery v0.28.8 // indirect
	k8s.io/utils v0.0.0-20230406110748-d93618cff8a2 // indirect
	modernc.org/libc v1.73.4 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

// 本地 patch：把 Wails v3 popup menu 显示路径的 fatal() 降级为 error()。
// 根因：Windows 25H2 Insider 会话上下文下 GetCursorPos 启动时返回失败，
// Wails alpha2.115 直接 fatal 退出应用。官方未修（见 PR #5228/#5234 讨论）。
replace github.com/wailsapp/wails/v3 => ./_wails-patch
