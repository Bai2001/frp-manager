// Package portprobe 通过尝试监听判断 TCP/UDP 端口在本地是否可用。
package portprobe

import (
	"fmt"
	"net"
)

// TCPAvailable 判断本地 TCP port 是否可被监听（即可用作 frps 远程端口）。
// 绑定 127.0.0.1 与测试占用保持一致，避免 :port 在某些系统上与 127.0.0.1:port 不冲突。
func TCPAvailable(port int) (bool, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false, nil // 监听失败视为被占用
	}
	_ = ln.Close()
	return true, nil
}

// UDPAvailable 判断本地 UDP port 是否可被监听。
func UDPAvailable(port int) (bool, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		return false, nil
	}
	_ = pc.Close()
	return true, nil
}
