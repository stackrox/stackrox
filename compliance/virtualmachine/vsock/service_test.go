//go:build linux
// +build linux

package vsock

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestServiceStartStopNoClient(t *testing.T) {
	s := NewService(12345)
	r, err := s.Start()
	if err != nil {
		t.Skipf("kernel may not support AF_VSOCK here: %v", err)
	}
	if r == nil {
		t.Fatalf("expected runner")
	}
	if err := s.Stop(); err != nil {
		t.Fatalf("stop error: %v", err)
	}
}

func TestRunWithContextCancel(t *testing.T) {
	s := NewService(12346)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// No client will connect; we only verify cancellation path returns.
	h := ConnectionHandlerFunc(func(conn net.Conn) error { return nil })
	_ = s.RunWithContext(ctx, h)
}
