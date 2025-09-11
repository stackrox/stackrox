package vsock

import (
	"net"
	"testing"
)

func TestAddrImplementsNetAddr(t *testing.T) {
	var _ net.Addr = (*Addr)(nil)
}

func TestAddrCIDPortAndString(t *testing.T) {
	a := &Addr{cid: 42, port: 5555}
	if a.CID() != 42 {
		t.Fatalf("expected CID 42, got %d", a.CID())
	}
	if a.Port() != 5555 {
		t.Fatalf("expected Port 5555, got %d", a.Port())
	}
	if got := a.Network(); got != "vsock" {
		t.Fatalf("expected network 'vsock', got %q", got)
	}
	if got := a.String(); got != "42:5555" {
		t.Fatalf("expected string '42:5555', got %q", got)
	}
}
