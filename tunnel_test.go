package ctunnel

import (
	"bytes"
	"crypto/rand"
	"io"
	"net"
	"testing"
	"time"
)

func TestOpenTunnel(t *testing.T) {
	a1, a2 := net.Pipe()
	b1, b2 := net.Pipe()

	go func() {
		err := OpenTunnel(a2, b2, time.Second*5)
		if err != nil {
			t.Error(err)
		}
	}()

	data := make([]byte, 8*1024)
	buf := make([]byte, 8*1024)
	rand.Read(data)

	_, err := a1.Write(data)
	if err != nil {
		t.Fatal(err)
	}

	_, err = io.ReadFull(b1, buf)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, buf) {
		t.Fatal("broken data")
	}
}

func TestOpenTunnelTimeout(t *testing.T) {
	_, a2 := net.Pipe()
	_, b2 := net.Pipe()

	start := time.Now()
	err := OpenTunnel(a2, b2, time.Second*1)
	if err == nil {
		t.Error("OpenTunnel should return a timeout err")
	}
	if time.Since(start) < time.Second {
		t.Error("OpenTunnel returned too early")
	}
	if time.Since(start) > time.Millisecond*1200 {
		t.Error("OpenTunnel returned too slow")
	}
}

func TestOpenTunnelErrIO(t *testing.T) {
	_, a2 := net.Pipe()
	_, b2 := net.Pipe()

	a2.Close()
	start := time.Now()
	err := OpenTunnel(a2, b2, time.Second*1)
	if err == nil {
		t.Error("OpenTunnel should return a closed err")
	}
	if time.Since(start) > time.Millisecond*50 {
		t.Error("OpenTunnel returned too slow")
	}
}
