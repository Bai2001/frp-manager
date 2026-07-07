package portprobe

import (
	"net"
	"testing"
)

func TestTCPAvailable_freePort(t *testing.T) {
	// 任意未占用端口应判定为可用
	got, err := TCPAvailable(59999)
	if err != nil {
		t.Fatalf("TCPAvailable: %v", err)
	}
	if !got {
		t.Errorf("TCPAvailable(59999) = false, want true")
	}
}

func TestTCPAvailable_occupiedPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	got, err := TCPAvailable(port)
	if err != nil {
		t.Fatalf("TCPAvailable: %v", err)
	}
	if got {
		t.Errorf("TCPAvailable(%d) = true, want false (端口已被占用)", port)
	}
}

func TestUDPAvailable_freePort(t *testing.T) {
	got, err := UDPAvailable(59998)
	if err != nil {
		t.Fatalf("UDPAvailable: %v", err)
	}
	if !got {
		t.Errorf("UDPAvailable(59998) = false, want true")
	}
}

func TestUDPAvailable_occupiedPort(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	port := pc.LocalAddr().(*net.UDPAddr).Port
	got, err := UDPAvailable(port)
	if err != nil {
		t.Fatalf("UDPAvailable: %v", err)
	}
	if got {
		t.Errorf("UDPAvailable(%d) = true, want false", port)
	}
}
