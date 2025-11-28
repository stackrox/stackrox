// Package vsock provides utilities for working with vsock (VM sockets) connections.
// It handles vsock-specific operations like creating listeners and extracting context IDs.
package vsock

import (
	"fmt"
	"net"

	"github.com/mdlayher/vsock"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
)

// ExtractVsockCIDFromConnection extracts and validates the vsock context ID.
// Rejects CIDs ≤2 which are reserved (0=ANY, 1=LOCAL, 2=HOST per vsock spec).
func ExtractVsockCIDFromConnection(conn net.Conn) (uint32, error) {
	remoteAddr, ok := conn.RemoteAddr().(*vsock.Addr)
	if !ok {
		return 0, fmt.Errorf("failed to extract remote address from vsock connection: unexpected type %T, value: %v",
			conn.RemoteAddr(), conn.RemoteAddr())
	}

	// Reject invalid values according to the vsock spec (https://www.man7.org/linux/man-pages/man7/vsock.7.html)
	if remoteAddr.ContextID <= 2 {
		return 0, fmt.Errorf("received an invalid vsock context ID: %d (values <=2 are reserved)", remoteAddr.ContextID)
	}

	return remoteAddr.ContextID, nil
}

// NewListener creates a vsock listener on the host context ID (vsock.Host) using the port
// from VirtualMachinesVsockPort env var. Caller must close the returned listener.
func NewListener() (net.Listener, error) {
	port := env.VirtualMachinesVsockPort.IntegerSetting()
	listener, err := vsock.ListenContextID(vsock.Host, uint32(port), nil)
	if err != nil {
		return nil, errors.Wrapf(err, "listening on vsock port %d", port)
	}
	return listener, nil
}

// DialHost establishes a vsock connection to the host context ID using the configured port.
func DialHost() (net.Conn, error) {
	port := env.VirtualMachinesVsockPort.IntegerSetting()
	conn, err := vsock.Dial(vsock.Host, uint32(port), nil)
	if err != nil {
		return nil, errors.Wrapf(err, "dialing host vsock port %d", port)
	}
	return conn, nil
}

// DialLocal establishes a vsock loopback connection using the configured port.
// This is intended for local load testing when both client and server run on the same host.
func DialLocal() (net.Conn, error) {
	port := env.VirtualMachinesVsockPort.IntegerSetting()
	conn, err := vsock.Dial(vsock.Local, uint32(port), nil)
	if err != nil {
		return nil, errors.Wrapf(err, "dialing local vsock port %d", port)
	}
	return conn, nil
}
