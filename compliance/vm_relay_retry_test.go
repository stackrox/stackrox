package compliance

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/compliance/virtualmachines/relay/stream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateVMRelayStreamWithRetry_SucceedsAfterFailures(t *testing.T) {
	failCount := 2
	attempts := atomic.Int32{}

	createStream := func() (*stream.VsockIndexReportStream, error) {
		n := attempts.Add(1)
		// Simulate failure cases: return an error until the success case is reached.
		if n <= int32(failCount) {
			return nil, errors.New("vsock not available")
		}
		// Success case: use a TCP listener as a stand-in for vsock (test-only).
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { _ = listener.Close() })
		s, err := stream.NewWithListener(listener)
		require.NoError(t, err)
		return s, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reportStream, err := createVMRelayStreamWithRetry(ctx, createStream)
	require.NoError(t, err)
	require.NotNil(t, reportStream)
	assert.GreaterOrEqual(t, attempts.Load(), int32(failCount+1), "should have retried at least %d times before success", failCount+1)
}

func TestCreateVMRelayStreamWithRetry_CancellationStopsRetryPromptly(t *testing.T) {
	attempts := atomic.Int32{}
	createStream := func() (*stream.VsockIndexReportStream, error) {
		attempts.Add(1)
		return nil, errors.New("vsock not available")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := createVMRelayStreamWithRetry(ctx, createStream)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.LessOrEqual(t, attempts.Load(), int32(1), "should not retry after cancellation")
}

func TestCreateVMRelayStreamWithRetry_SucceedsImmediately(t *testing.T) {
	createStream := func() (*stream.VsockIndexReportStream, error) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { _ = listener.Close() })
		return stream.NewWithListener(listener)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reportStream, err := createVMRelayStreamWithRetry(ctx, createStream)
	require.NoError(t, err)
	require.NotNil(t, reportStream)
}
