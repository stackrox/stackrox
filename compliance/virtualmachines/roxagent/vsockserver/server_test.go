package vsockserver

import (
	"context"
	"io"
	"net"
	"testing"

	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/vsockframing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// TestServeAcceptLoop verifies that the weighted semaphore (maxConcurrentConns=1)
// rejects a second connection while the first is still being handled, and that
// cancelling the context drains gracefully.
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

	// Second connection: should be rejected (semaphore full).
	conn2, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)
	buf := make([]byte, 1)
	_, err = conn2.Read(buf)
	assert.ErrorIs(t, err, io.EOF, "second connection should be closed immediately")

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
