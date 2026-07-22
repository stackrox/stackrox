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
//
// With maxConcurrentConns=1, the stalled peer is accepted first and holds
// the semaphore for the duration of its (never-completing) handshake, so
// the second peer always takes the rejectConn path and receives
// ERROR_CODE_BUSY. That is enough to prove Accept() kept running: a blocked
// accept loop would never handshake the second peer or write BUSY.
// The client must only ReadFrame here - writing a request races rejectConn's
// close-after-BUSY and flakes with broken pipe on Linux.
func TestServeAcceptLoop_StalledHandshakeDoesNotBlockOtherConnections(t *testing.T) {
	handler := NewHandler(&ReportCache{}, "test")
	srv := NewServer(handler, &tls.Config{Certificates: []tls.Certificate{testServerCert(t)}})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan struct{})
	go func() { srv.Serve(ctx, ln); close(serveDone) }()
	defer func() {
		cancel()
		<-serveDone
	}()

	// A stalled peer: opens the TCP connection but never sends a TLS
	// ClientHello (or anything else). That peer wins the single semaphore
	// slot; serveConn then blocks in HandshakeContext off the accept loop.
	stalled, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)
	// Close before cancel (defer LIFO) so the handshake goroutine unblocks
	// without waiting out connDeadline.
	defer func() { _ = stalled.Close() }()

	// Second peer on the test goroutine (same style as TestServeAcceptLoop).
	// NetDialer.Timeout bounds dial+TLS handshake: if Accept were blocked,
	// TCP can still complete from the backlog, but the handshake would hang.
	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: 5 * time.Second},
		Config:    &tls.Config{InsecureSkipVerify: true},
	}
	clientConn, err := dialer.DialContext(ctx, "tcp", ln.Addr().String())
	require.NoError(t, err, "second peer should be accepted promptly; accept loop may be blocked")
	defer func() { _ = clientConn.Close() }()

	require.NoError(t, clientConn.SetDeadline(time.Now().Add(5*time.Second)))
	respData, err := vsockframing.ReadFrame(clientConn, 1<<20)
	require.NoError(t, err, "rejected peer should receive framed BUSY before the server closes")
	var resp pb.VMServiceResponse
	require.NoError(t, proto.Unmarshal(respData, &resp))
	require.NotNil(t, resp.GetError())
	assert.Equal(t, pb.ErrorCode_ERROR_CODE_BUSY, resp.GetError().GetCode())
}
