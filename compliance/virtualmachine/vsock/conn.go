package vsock

import (
	"net"
	"time"

	"golang.org/x/sys/unix"
)

// vsockConn represents a vsock connection
type vsockConn struct {
	fd         int
	localAddr  *vsockAddr
	remoteAddr *vsockAddr
	listener   *vsockListener
}

// Read reads data from the connection
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

// Write writes data to the connection
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

// Close closes the connection
func (c *vsockConn) Close() error {
	// Remove from listener's connection tracking
	if c.listener != nil {
		c.listener.connMu.Lock()
		delete(c.listener.connections, c.fd)
		c.listener.connMu.Unlock()
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

// LocalAddr returns the local address
func (c *vsockConn) LocalAddr() net.Addr {
	return c.localAddr
}

// RemoteAddr returns the remote address
func (c *vsockConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

// SetDeadline sets the read and write deadlines
func (c *vsockConn) SetDeadline(t time.Time) error {
	// vsock doesn't support deadlines directly, this is a no-op
	return nil
}

// SetReadDeadline sets the read deadline
func (c *vsockConn) SetReadDeadline(t time.Time) error {
	// vsock doesn't support deadlines directly, this is a no-op
	return nil
}

// SetWriteDeadline sets the write deadline
func (c *vsockConn) SetWriteDeadline(t time.Time) error {
	// vsock doesn't support deadlines directly, this is a no-op
	return nil
}
