package vsockserver

import (
	"context"
	"crypto/tls"
	"net"
	"testing"
	"time"

	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/vsockframing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// TestServeAcceptLoop verifies that the weighted semaphore (maxConcurrentConns=1)
// rejects a second connection with an ERROR_CODE_BUSY response while the first
// is still being handled, and that cancelling the context drains gracefully.
func TestServeAcceptLoop(t *testing.T) {
	handler := NewHandler(&ReportCache{}, "test")
	srv := NewServer(handler, nil)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan struct{})
	go func() { srv.Serve(ctx, ln); close(serveDone) }()

	// First connection: hold the semaphore by not sending data yet.
	conn1, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)

	// Second connection: should be rejected (semaphore full) with a BUSY response.
	conn2, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)
	defer func() { _ = conn2.Close() }()
	busyData, err := vsockframing.ReadFrame(conn2, 1<<20)
	require.NoError(t, err, "rejected connection should still receive a framed response before closing")
	var busyResp pb.VMServiceResponse
	require.NoError(t, proto.Unmarshal(busyData, &busyResp))
	require.NotNil(t, busyResp.GetError())
	assert.Equal(t, pb.ErrorCode_ERROR_CODE_BUSY, busyResp.GetError().GetCode())

	// Complete first connection: send a request and read NOT_READY response.
	req, _ := proto.Marshal(&pb.VMServiceRequest{Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{}}})
	require.NoError(t, vsockframing.WriteFrame(conn1, req))
	respData, err := vsockframing.ReadFrame(conn1, 1<<20)
	require.NoError(t, err)
	var resp pb.VMServiceResponse
	require.NoError(t, proto.Unmarshal(respData, &resp))
	assert.Equal(t, pb.ErrorCode_ERROR_CODE_NOT_READY, resp.GetError().GetCode())
	_ = conn1.Close()

	// Graceful shutdown.
	cancel()
	<-serveDone
}

// TestServeAcceptLoop_StalledHandshakeDoesNotBlockOtherConnections is a
// regression test for a bug where the TLS handshake ran inline in Serve's
// accept loop: a peer that connects and never completes (or never starts)
// the handshake blocked Accept() from ever being called again, starving
// every other connection - including the legitimate one - for as long as
// the stalled peer stayed connected. The handshake must run off the accept
// loop so Accept() only ever blocks on Accept() itself.
func TestServeAcceptLoop_StalledHandshakeDoesNotBlockOtherConnections(t *testing.T) {
	handler := NewHandler(&ReportCache{}, "test")
	srv := NewServer(handler, &tls.Config{Certificates: []tls.Certificate{testServerCert(t)}})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan struct{})
	go func() { srv.Serve(ctx, ln); close(serveDone) }()

	// A stalled peer: opens the TCP connection but never sends a TLS
	// ClientHello (or anything else). Before the fix, the server's
	// HandshakeContext call for this connection would block Serve's single
	// accept loop indefinitely, since it has no deadline of its own.
	stalled, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)
	defer func() { _ = stalled.Close() }()

	// A well-behaved peer must still be accepted and served promptly,
	// despite the stalled connection still being open.
	// require inside a non-test goroutine only stops that goroutine (via
	// runtime.Goexit), not the test itself, so failures here use assert with
	// an early return instead.
	done := make(chan struct{})
	go func() {
		defer close(done)
		clientConn, dialErr := tls.Dial("tcp", ln.Addr().String(), &tls.Config{InsecureSkipVerify: true})
		if !assert.NoError(t, dialErr) {
			return
		}
		defer func() { _ = clientConn.Close() }()

		req, _ := proto.Marshal(&pb.VMServiceRequest{Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{}}})
		if !assert.NoError(t, vsockframing.WriteFrame(clientConn, req)) {
			return
		}
		respData, readErr := vsockframing.ReadFrame(clientConn, 1<<20)
		if !assert.NoError(t, readErr) {
			return
		}
		var resp pb.VMServiceResponse
		if !assert.NoError(t, proto.Unmarshal(respData, &resp)) {
			return
		}
		assert.NotNil(t, resp.GetError(), "expected NOT_READY, but any response at all proves the loop wasn't blocked")
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("well-behaved connection was not served promptly; accept loop is likely blocked by the stalled peer")
	}

	// Unblock the stalled connection's server-side handshake goroutine
	// immediately, rather than waiting out connDeadline, so the test stays fast.
	_ = stalled.Close()
	cancel()
	<-serveDone
}
