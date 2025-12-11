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
// In test mode, returns vsock.Local to enable loopback testing.
func ExtractVsockCIDFromConnection(conn net.Conn) (uint32, error) {
	if env.IsVMTestModeEnabled() {
		return vsock.Local, nil
	}
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

// NewListener creates a vsock listener using the port from VirtualMachinesVsockPort env var.
// In test mode, binds to vsock.Local for loopback testing. Otherwise binds to vsock.Host for production.
// Caller must close the returned listener.
func NewListener() (net.Listener, error) {
	port := env.VirtualMachinesVsockPort.IntegerSetting()

	var contextID uint32 = vsock.Host
	if env.IsVMTestModeEnabled() {
		contextID = vsock.Local
	}

	listener, err := vsock.ListenContextID(contextID, uint32(port), nil)
	if err != nil {
		return nil, errors.Wrapf(err, "listening on vsock port %d", port)
	}
	return listener, nil
}
