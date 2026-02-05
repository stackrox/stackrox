package vsock

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/stackrox/rox/pkg/logging"
	"golang.org/x/sys/unix"
	"google.golang.org/protobuf/proto"
)

var (
	log = logging.LoggerForModule()
)

// Client represents a VSOCK client connection
type Client struct {
	conn net.Conn
}

// Connect establishes a VSOCK connection to the host
func Connect() (*Client, error) {
	// Create VSOCK socket
	fd, err := unix.Socket(unix.AF_VSOCK, unix.SOCK_STREAM, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to create VSOCK socket: %w", err)
	}

	// Connect to host (CID 2) on port 818 (matching Rust implementation)
	// The port should match what the VSOCK listener is expecting
	addr := &unix.SockaddrVM{
		CID:  unix.VMADDR_CID_HOST, // Connect to host
		Port: 818,                  // VSOCK port (matches Rust vm_agent)
	}

	if err := unix.Connect(fd, addr); err != nil {
		if closeErr := unix.Close(fd); closeErr != nil {
			log.Warnf("Failed to close socket fd after connect error: %v", closeErr)
		}
		return nil, fmt.Errorf("failed to connect to VSOCK host: %w", err)
	}

	// Create a net.Conn wrapper for the file descriptor
	conn, err := createNetConn(fd)
	if err != nil {
		if closeErr := unix.Close(fd); closeErr != nil {
			log.Warnf("Failed to close socket fd after createNetConn error: %v", closeErr)
		}
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	return &Client{conn: conn}, nil
}

// SendData sends protobuf-encoded data over VSOCK with length prefix
func (c *Client) SendData(data []byte) error {
	// Set write timeout
	if err := c.conn.SetWriteDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	// Send 4-byte length prefix (little-endian)
	header := make([]byte, 4)
	binary.LittleEndian.PutUint32(header, uint32(len(data)))

	if _, err := c.conn.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Send the data
	if _, err := c.conn.Write(data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	// Read acknowledgment (4 bytes)
	ack := make([]byte, 4)
	if err := c.conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return fmt.Errorf("failed to set read deadline: %w", err)
	}

	if _, err := c.conn.Read(ack); err != nil {
		return fmt.Errorf("failed to read acknowledgment: %w", err)
	}

	// Check acknowledgment (0 = success)
	ackValue := binary.LittleEndian.Uint32(ack)
	if ackValue != 0 {
		return fmt.Errorf("received error acknowledgment: %d", ackValue)
	}

	return nil
}

// SendProtobuf sends a protobuf message over VSOCK
func (c *Client) SendProtobuf(msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal protobuf: %w", err)
	}

	return c.SendData(data)
}

// Close closes the VSOCK connection
func (c *Client) Close() error {
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return fmt.Errorf("failed to close VSOCK connection: %w", err)
		}
	}
	return nil
}

// createNetConn creates a net.Conn from a file descriptor
func createNetConn(fd int) (net.Conn, error) {
	// Convert file descriptor to *os.File
	file := os.NewFile(uintptr(fd), "vsock")
	if file == nil {
		return nil, errors.New("failed to create file from fd")
	}

	// Create net.Conn from file
	conn, err := net.FileConn(file)
	if err != nil {
		if closeErr := file.Close(); closeErr != nil {
			log.Warnf("Failed to close file after FileConn error: %v", closeErr)
		}
		return nil, fmt.Errorf("failed to create connection from file: %w", err)
	}

	// Close the file (net.Conn takes ownership)
	if err := file.Close(); err != nil {
		log.Warnf("Failed to close file after successful FileConn creation: %v", err)
	}

	return conn, nil
}
