package cmd

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNotifySystemdReady_NoSocketIsNoop covers running without systemd
// (NOTIFY_SOCKET unset): readiness notify must not error.
func TestNotifySystemdReady_NoSocketIsNoop(t *testing.T) {
	t.Setenv("NOTIFY_SOCKET", "")
	require.NoError(t, notifySystemdReady())
}

// TestNotifySystemdReady_WritesReady covers the sd_notify protocol against a
// fake NOTIFY_SOCKET: READY=1 must be written as a unixgram datagram.
func TestNotifySystemdReady_WritesReady(t *testing.T) {
	// Prefer a short path under /tmp: macOS AF_UNIX sun_path is small, and
	// t.TempDir() paths are often too long to bind.
	f, err := os.CreateTemp("", "rox-sdn-*.sock")
	require.NoError(t, err)
	sockPath := f.Name()
	require.NoError(t, f.Close())
	require.NoError(t, os.Remove(sockPath))
	t.Cleanup(func() { _ = os.Remove(sockPath) })

	addr := &net.UnixAddr{Name: sockPath, Net: "unixgram"}
	ln, err := net.ListenUnixgram("unixgram", addr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	t.Setenv("NOTIFY_SOCKET", sockPath)

	require.NoError(t, notifySystemdReady())

	_ = ln.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 64)
	n, _, err := ln.ReadFromUnix(buf)
	require.NoError(t, err)
	assert.Equal(t, "READY=1", string(buf[:n]))
}
