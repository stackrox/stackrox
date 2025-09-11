package vsock

import (
	"errors"
	"net"
	"syscall"

	pkgerrors "github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/sys/unix"
)

var (
	log = logging.LoggerForModule()
)

// vsockListener implements net.Listener for vsock connections
type vsockListener struct {
	fd          int
	port        uint32
	mu          sync.Mutex
	closed      bool
	connections map[int]*vsockConn
	connMu      sync.RWMutex
	stopper     concurrency.Stopper
}

func newListener(fd int, port uint32) *vsockListener {
	return &vsockListener{
		fd:      fd,
		port:    port,
		stopper: concurrency.NewStopper(),
	}
}

// Accept accepts incoming vsock connections
func (l *vsockListener) Accept() (net.Conn, error) {

	connFd, clientAddr, err := unix.Accept(l.fd)
	if err != nil {
		return nil, &net.OpError{
			Op:  "accept",
			Net: "vsock",
			Err: err,
		}
	}

	// Configure the accepted connection
	if err := configureConnection(connFd); err != nil {
		err := unix.Close(connFd)
		if err != nil {
			log.Errorf("failed to close connection: %v", err)
		}
		return nil, pkgerrors.Wrap(err, "failed to configure connection")
	}

	// Extract address information
	var remoteAddr *Addr
	if vmAddr, ok := clientAddr.(*unix.SockaddrVM); ok {
		remoteAddr = newAddr(vmAddr)
	} else {
		// Fallback if we can't get the remote address
		remoteAddr = &Addr{cid: 0, port: 0}
	}

	// Get local address
	localAddr := &Addr{
		cid:  unix.VMADDR_CID_HOST,
		port: l.port,
	}

	// Create connection wrapper
	conn := &vsockConn{
		fd:         connFd,
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
		listener:   l,
	}

	// Track the connection
	concurrency.WithLock(&l.connMu, func() {
		if l.connections == nil {
			l.connections = make(map[int]*vsockConn)
		}
		l.connections[connFd] = conn
	})

	return conn, nil
}

// Close closes the listener and all active connections
func (l *vsockListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	l.closed = true

	// Close all active connections
	var connections []*vsockConn
	concurrency.WithRLock(&l.connMu, func() {
		connections = make([]*vsockConn, 0, len(l.connections))
		for _, conn := range l.connections {
			connections = append(connections, conn)
		}
	})

	errList := errorhelpers.NewErrorList("vsock listener close")
	for _, conn := range connections {
		if err := conn.Close(); err != nil {
			errList.AddError(err)
		}
	}

	if l.fd >= 0 {
		if err := unix.Close(l.fd); err != nil {
			errList.AddError(err)
		}
	}

	return errList.ToError()
}

// Addr returns the listener's network address
func (l *vsockListener) Addr() net.Addr {
	return &Addr{
		cid:  unix.VMADDR_CID_HOST,
		port: l.port,
	}
}

// configureConnection sets up options for an accepted connection
func configureConnection(fd int) error {
	// Enable keep-alive for the connection
	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_KEEPALIVE, 1); err != nil {
		// Keep-alive might not be supported, log but don't fail
		// In a real implementation, you might want to log this
		log.Errorf("failed to set keep-alive: %v", err)
	}

	// Set socket to non-blocking mode initially, but can be changed later
	if err := unix.SetNonblock(fd, false); err != nil {
		return pkgerrors.Wrap(err, "failed to set connection to non-blocking mode")
	}

	return nil
}

// AcceptLoop continuously accepts connections and handles them using the provided handler
func (l *vsockListener) AcceptLoop(handler ConnectionHandler) error {
	if handler == nil {
		return errors.New("connection handler cannot be nil")
	}
	log.Infof("AcceptLoop started")

	for {
		conn, err := l.Accept()
		if err != nil {
			// Check if this is a temporary error
			if netErr, ok := err.(*net.OpError); ok {
				if netErr.Temporary() {
					continue // Retry on temporary errors
				}
				// Check if the listener was closed
				if netErr.Err == net.ErrClosed || netErr.Err == syscall.EINVAL {
					return nil // Normal shutdown
				}
			}
			log.Errorf("failed to accept connection: %v", err)
			continue
		}
		log.Infof("Accepted connection")
		// gualvare push to channel here
		// Handle connection in a goroutine
		go func(c net.Conn) {
			defer func() {
				if closeErr := c.Close(); closeErr != nil {
					log.Errorf("failed to close connection: %v", closeErr)
					// In a real implementation, you might want to log this error
				}
			}()

			if err := handler.Handle(c); err != nil {
				log.Errorf("failed to handle connection: %v", err)
				// In a real implementation, you might want to log this error
			}
		}(conn)
	}
}
