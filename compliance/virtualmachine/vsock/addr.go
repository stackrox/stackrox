package vsock

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// Addr represents a vsock address and implements net.Addr
type Addr struct {
	cid  uint32
	port uint32
}

// Network returns the network type
func (a *Addr) Network() string {
	return "vsock"
}

// String returns the string representation of the address
func (a *Addr) String() string {
	return fmt.Sprintf("%d:%d", a.cid, a.port)
}

// CID returns the context ID (CID)
func (a *Addr) CID() uint32 { return a.cid }

// Port returns the port number
func (a *Addr) Port() uint32 { return a.port }

// newAddr creates a new vsock address from unix.SockaddrVM
func newAddr(addr *unix.SockaddrVM) *Addr {
	return &Addr{
		cid:  addr.CID,
		port: addr.Port,
	}
}
