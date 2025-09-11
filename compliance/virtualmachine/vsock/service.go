package vsock

import (
	"context"
	"errors"
	"net"

	pkgerrors "github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/sys/unix"
)

// Service represents a vsock server that can accept connections from virtual machines
type Service struct {
	port     uint32
	fd       int
	listener *vsockListener
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

	log.Infof("Starting vsock server")
	if !s.closed {
		return nil, errors.New("server is already running")
	}

	// Create vsock socket
	fd, err := unix.Socket(unix.AF_VSOCK, unix.SOCK_STREAM, 0)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to create vsock socket")
	}
	log.Infof("Created vsock socket")

	s.fd = fd
	defer func() {
		// Close in case of error
		if err != nil {
			err := unix.Close(s.fd)
			if err != nil {
				log.Errorf("failed to close vsock socket: %v", err)
			}
		}
	}()

	// Configure socket options
	if err = s.configureSocket(); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to configure socket")
	}

	// Bind to host address with configured port
	addr := &unix.SockaddrVM{
		CID:  unix.VMADDR_CID_HOST,
		Port: s.port,
	}

	if err = unix.Bind(fd, addr); err != nil {
		return nil, pkgerrors.Wrapf(err, "failed to bind vsock socket to address %d port %d", addr.CID, addr.Port)
	}
	log.Infof("Bound vsock socket to port %d", s.port)

	// Start listening
	if err = unix.Listen(fd, unix.SOMAXCONN); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to listen on vsock socket")
	}

	// Create a net.Listener wrapper for easier connection handling
	s.listener = newListener(s.fd, s.port)
	log.Infof("Created vsock listener")

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
		return pkgerrors.Wrap(err, "failed to set SO_REUSEADDR")
	}

	// Set non-blocking mode for the listening socket
	if err := unix.SetNonblock(s.fd, false); err != nil {
		return pkgerrors.Wrap(err, "failed to set socket to non-blocking mode")
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
		return errors.New("connection handler cannot be nil")
	}
	if r == nil {
		return errors.New("vsock runner cannot be nil")
	}
	if r.listener == nil {
		return errors.New("vsock listener cannot be nil")
	}

	return r.listener.AcceptLoop(handler)
}

// RunWithContext starts the service, runs the accept loop, and stops when the context is done.
func (s *Service) RunWithContext(ctx context.Context, handler ConnectionHandler) error {
	runner, err := s.Start()
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- runner.Run(handler)
	}()

	select {
	case <-ctx.Done():
		_ = s.Stop()
		return nil
	case err := <-errCh:
		_ = s.Stop()
		return err
	}
}
