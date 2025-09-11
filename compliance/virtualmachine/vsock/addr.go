package vsock

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// vsockAddr represents a vsock address
type vsockAddr struct {
	cid  uint32
	port uint32
}

// Network returns the network type
func (a *vsockAddr) Network() string {
	return "vsock"
}

// String returns the string representation of the address
func (a *vsockAddr) String() string {
	return fmt.Sprintf("%d:%d", a.cid, a.port)
}

// newVsockAddr creates a new vsock address from unix.SockaddrVM
func newVsockAddr(addr *unix.SockaddrVM) *vsockAddr {
	return &vsockAddr{
		cid:  addr.CID,
		port: addr.Port,
	}
}
