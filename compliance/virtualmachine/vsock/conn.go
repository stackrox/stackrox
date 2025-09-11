package vsock

import (
	"net"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"golang.org/x/sys/unix"
)

type vsockConn struct {
	fd         int
	localAddr  *Addr
	remoteAddr *Addr
	listener   *vsockListener
}

func (c *vsockConn) Read(b []byte) (int, error) {
	n, err := unix.Read(c.fd, b)
	if err != nil {
		return 0, &net.OpError{
			Op:   "read",
			Net:  "vsock",
			Addr: c.remoteAddr,
			Err:  err,
		}
	}
	return n, nil
}

func (c *vsockConn) Write(b []byte) (int, error) {
	n, err := unix.Write(c.fd, b)
	if err != nil {
		return 0, &net.OpError{
			Op:   "write",
			Net:  "vsock",
			Addr: c.remoteAddr,
			Err:  err,
		}
	}
	return n, nil
}

func (c *vsockConn) Close() error {
	// Remove from listener's connection tracking
	if c.listener != nil {
		concurrency.WithLock(&c.listener.connMu, func() {
			delete(c.listener.connections, c.fd)
		})
	}

	err := unix.Close(c.fd)
	if err != nil {
		return &net.OpError{
			Op:   "close",
			Net:  "vsock",
			Addr: c.remoteAddr,
			Err:  err,
		}
	}
	return nil
}

func (c *vsockConn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *vsockConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *vsockConn) SetDeadline(t time.Time) error {
	// vsock doesn't support deadlines directly, this is a no-op
	return nil
}

func (c *vsockConn) SetReadDeadline(t time.Time) error {
	// vsock doesn't support deadlines directly, this is a no-op
	return nil
}

func (c *vsockConn) SetWriteDeadline(t time.Time) error {
	// vsock doesn't support deadlines directly, this is a no-op
	return nil
}
