package vsock

import (
	"fmt"
	"net"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

// vsockListener implements net.Listener for VSOCK sockets
type vsockListener struct {
	fd int
}

func (l *vsockListener) Accept() (net.Conn, error) {
	connFD, sa, err := unix.Accept(l.fd)
	if err != nil {
		return nil, err
	}

	// Extract peer address from sockaddr
	peerAddr, ok := sa.(*unix.SockaddrVM)
	if !ok {
		unix.Close(connFD)
		return nil, errors.New("expected VSOCK sockaddr")
	}

	// Get local address info
	localSA, err := unix.Getsockname(l.fd)
	if err != nil {
		unix.Close(connFD)
		return nil, errors.Wrap(err, "failed to get local address")
	}

	localAddr, ok := localSA.(*unix.SockaddrVM)
	if !ok {
		unix.Close(connFD)
		return nil, errors.New("expected VSOCK local sockaddr")
	}

	// Create custom VSOCK connection
	conn := newVSockConn(
		connFD,
		localAddr.CID, localAddr.Port,
		peerAddr.CID, peerAddr.Port,
	)

	return conn, nil
}

func (l *vsockListener) Close() error {
	return unix.Close(l.fd)
}

func (l *vsockListener) Addr() net.Addr {
	return &vsockAddr{}
}

// vsockAddr implements net.Addr for VSOCK addresses
type vsockAddr struct {
	cid  uint32
	port uint32
}

func (a *vsockAddr) Network() string {
	return "vsock"
}

func (a *vsockAddr) String() string {
	return fmt.Sprintf("vsock:%d:%d", a.cid, a.port)
}

// vsockConn implements net.Conn for VSOCK connections
type vsockConn struct {
	fd         int
	localAddr  *vsockAddr
	remoteAddr *vsockAddr
}

func newVSockConn(fd int, localCID, localPort, remoteCID, remotePort uint32) *vsockConn {
	return &vsockConn{
		fd: fd,
		localAddr: &vsockAddr{
			cid:  localCID,
			port: localPort,
		},
		remoteAddr: &vsockAddr{
			cid:  remoteCID,
			port: remotePort,
		},
	}
}

// Read implements net.Conn.Read
func (c *vsockConn) Read(b []byte) (int, error) {
	return unix.Read(c.fd, b)
}

// Write implements net.Conn.Write
func (c *vsockConn) Write(b []byte) (int, error) {
	return unix.Write(c.fd, b)
}

// Close implements net.Conn.Close
func (c *vsockConn) Close() error {
	return unix.Close(c.fd)
}

// LocalAddr implements net.Conn.LocalAddr
func (c *vsockConn) LocalAddr() net.Addr {
	return c.localAddr
}

// RemoteAddr implements net.Conn.RemoteAddr
func (c *vsockConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

// SetDeadline implements net.Conn.SetDeadline
func (c *vsockConn) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}

// SetReadDeadline implements net.Conn.SetReadDeadline
func (c *vsockConn) SetReadDeadline(t time.Time) error {
	timeout := timeToTimeval(t)
	return unix.SetsockoptTimeval(c.fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &timeout)
}

// SetWriteDeadline implements net.Conn.SetWriteDeadline
func (c *vsockConn) SetWriteDeadline(t time.Time) error {
	timeout := timeToTimeval(t)
	return unix.SetsockoptTimeval(c.fd, unix.SOL_SOCKET, unix.SO_SNDTIMEO, &timeout)
}

// GetPeerCID returns the peer's Context ID
func (c *vsockConn) GetPeerCID() uint32 {
	return c.remoteAddr.cid
}

// timeToTimeval converts a deadline time to a unix.Timeval timeout
func timeToTimeval(t time.Time) unix.Timeval {
	if t.IsZero() {
		// Zero time means no timeout
		return unix.Timeval{Sec: 0, Usec: 0}
	}

	// Convert absolute time to duration from now
	duration := time.Until(t)
	if duration <= 0 {
		// Already past deadline
		return unix.Timeval{Sec: 0, Usec: 1} // Very short timeout
	}

	return unix.Timeval{
		Sec:  int64(duration.Seconds()),
		Usec: int64((duration % time.Second) / time.Microsecond),
	}
}
