package vsock

import (
	"fmt"
	"net"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

// Service represents a vsock server that can accept connections from virtual machines
type Service struct {
	port     uint32
	fd       int
	listener *vsockListener
	lisener  net.Listener
	mu       sync.RWMutex
	closed   bool
}

// NewService creates a new vsock server with the given port
func NewService(port uint32) *Service {
	return &Service{
		port:   port,
		fd:     -1,
		closed: true,
	}
}

// ConnectionHandler defines the interface for handling vsock connections
type ConnectionHandler interface {
	Handle(conn net.Conn) error
}

// ConnectionHandlerFunc is an adapter to allow functions to be used as ConnectionHandler
type ConnectionHandlerFunc func(conn net.Conn) error

// Handle calls the function
func (f ConnectionHandlerFunc) Handle(conn net.Conn) error {
	return f(conn)
}

// Start creates, configures and binds the vsock socket, then starts listening
func (s *Service) Start() (*vsockRunner, error) {
	var err error
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.closed {
		return nil, fmt.Errorf("server is already running")
	}

	// Create vsock socket
	fd, err := unix.Socket(unix.AF_VSOCK, unix.SOCK_STREAM, 0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create vsock socket")
	}

	s.fd = fd
	defer func() {
		// Close in case of error
		if err != nil {
			unix.Close(s.fd)
		}
	}()

	// Configure socket options
	if err = s.configureSocket(); err != nil {
		return nil, errors.Wrap(err, "failed to configure socket")
	}

	// Bind to host address with configured port
	addr := &unix.SockaddrVM{
		CID:  unix.VMADDR_CID_HOST,
		Port: s.port,
	}

	if err = unix.Bind(fd, addr); err != nil {
		return nil, errors.Wrapf(err, "failed to bind vsock socket to port %d", s.port)
	}

	// Start listening
	if err = unix.Listen(fd, unix.SOMAXCONN); err != nil {
		return nil, errors.Wrap(err, "failed to listen on vsock socket")
	}

	// Create a net.Listener wrapper for easier connection handling
	s.listener = newListener(s.fd, s.port)

	s.closed = false
	return &vsockRunner{
		listener: s.listener,
	}, nil
}

// Stop stops the server and closes the socket
func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	var err error
	if s.listener != nil {
		err = s.listener.Close()
		s.listener = nil
	}

	if s.fd >= 0 {
		if closeErr := unix.Close(s.fd); closeErr != nil && err == nil {
			err = closeErr
		}
		s.fd = -1
	}

	s.closed = true
	return err
}

// configureSocket sets socket options for optimal performance
func (s *Service) configureSocket() error {
	// Set SO_REUSEADDR to allow quick restart of the server
	if err := unix.SetsockoptInt(s.fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
		return errors.Wrap(err, "failed to set SO_REUSEADDR")
	}

	// Set non-blocking mode for the listening socket
	if err := unix.SetNonblock(s.fd, false); err != nil {
		return errors.Wrap(err, "failed to set socket to blocking mode")
	}

	return nil
}

// vsockRunner implements VsockRunner interface
type vsockRunner struct {
	listener *vsockListener
}

// Run starts accepting connections and handles them using the provided handler
func (r *vsockRunner) Run(handler ConnectionHandler) error {
	if handler == nil {
		return fmt.Errorf("connection handler cannot be nil")
	}

	return r.listener.AcceptLoop(handler)
}
