package vm_relay

import (
	"context"
	"net"
	"time"

	"github.com/mdlayher/vsock"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

// Handler defines the interface for handling individual connections
type Handler interface {
	// HandleConnection processes a single connection and returns data
	HandleConnection(ctx context.Context, conn net.Conn) (interface{}, error)
}

// VsockServer defines the interface for a vsock server
type VsockServer interface {
	// Run starts the server loop
	Run(ctx context.Context, handler Handler, resultChan chan<- interface{})
	// Close closes the server
	Close() error
}

// vsockServerImpl implements VsockServer for handling vsock connections
type vsockServerImpl struct {
	config   *VsockServerConfig
	listener net.Listener
}

// VsockServerConfig holds configuration for the vsock server
type VsockServerConfig struct {
	Port        uint32
	ReadTimeout time.Duration
}

// DefaultVsockServerConfig returns default vsock server configuration
func DefaultVsockServerConfig() *VsockServerConfig {
	return &VsockServerConfig{
		Port:        1234,
		ReadTimeout: 20 * time.Second,
	}
}

// NewVsockServer creates a new vsock server with default configuration
func NewVsockServer() VsockServer {
	config := DefaultVsockServerConfig()
	return &vsockServerImpl{
		config: config,
	}
}

// NewVsockServerWithConfig creates a new vsock server with custom configuration
func NewVsockServerWithConfig(config *VsockServerConfig) VsockServer {
	return &vsockServerImpl{
		config: config,
	}
}

// NewVsockServerWithPort creates a new vsock server with a specific port
func NewVsockServerWithPort(port uint32) VsockServer {
	config := DefaultVsockServerConfig()
	config.Port = port
	return &vsockServerImpl{
		config: config,
	}
}

// Run starts the vsock server loop that listens for connections and processes them
func (s *vsockServerImpl) Run(ctx context.Context, handler Handler, resultChan chan<- interface{}) {
	log.Infof("Starting server on port %d", s.config.Port)

	// Create vsock listener directly
	listener, err := vsock.ListenContextID(vsock.Host, s.config.Port, nil)
	if err != nil {
		log.Errorf("Failed to create vsock listener: %v", err)
		return
	}
	defer func() {
		if err := listener.Close(); err != nil {
			log.Errorf("Failed to close listener: %v", err)
		}
	}()
	s.listener = listener

	log.Infof("Server listener created, waiting for connections...")

	// Accept connections in a loop
	for {
		select {
		case <-ctx.Done():
			log.Infof("Server stopping")
			return
		default:
			// Accept connection
			conn, err := listener.Accept()
			if err != nil {
				log.Errorf("Failed to accept connection: %v", err)
				continue
			}

			// Handle connection in a goroutine
			go s.handleConnection(ctx, conn, handler, resultChan)
		}
	}
}

// Close closes the vsock server
func (s *vsockServerImpl) Close() error {
	log.Infof("Closing server")
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// handleConnection handles a single connection using the provided handler
func (s *vsockServerImpl) handleConnection(ctx context.Context, conn net.Conn, handler Handler, resultChan chan<- interface{}) {
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("Failed to close connection: %v", err)
		}
	}()

	log.Debugf("Handling connection from %s", conn.RemoteAddr())

	// Set read timeout for this connection
	if err := conn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout)); err != nil {
		log.Errorf("Failed to set read deadline: %v", err)
		return
	}

	// Use the handler to process the connection
	result, err := handler.HandleConnection(ctx, conn)
	if err != nil {
		log.Errorf("Handler failed to process connection: %v", err)
		return
	}

	// Send the result to the channel
	select {
	case resultChan <- result:
		log.Debugf("Result sent to channel successfully")
	case <-ctx.Done():
		log.Debugf("Context cancelled while sending result")
		return
	}
}
