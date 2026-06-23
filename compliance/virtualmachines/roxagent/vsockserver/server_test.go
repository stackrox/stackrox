package vsockserver

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestServer_ServesReport(t *testing.T) {
	report := &v1.VMReport{
		IndexReport: &v1.IndexReport{
			VsockCid: "42",
			IndexV4: &v4.IndexReport{
				Success: true,
			},
		},
	}

	srv := NewServer()
	srv.SetReport(report)

	clientConn, serverConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- srv.HandleConn(ctx, serverConn)
	}()

	data, err := io.ReadAll(clientConn)
	require.NoError(t, err)

	got := &v1.VMReport{}
	require.NoError(t, proto.Unmarshal(data, got))
	assert.Equal(t, "42", got.GetIndexReport().GetVsockCid())
	assert.True(t, got.GetIndexReport().GetIndexV4().GetSuccess())

	require.NoError(t, <-done)
}

func TestServer_NoReport(t *testing.T) {
	srv := NewServer()

	clientConn, serverConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := srv.HandleConn(ctx, serverConn)
	assert.ErrorContains(t, err, "no report available")
}

func TestServer_AcceptLoop(t *testing.T) {
	report := &v1.VMReport{
		IndexReport: &v1.IndexReport{
			VsockCid: "7",
			IndexV4:  &v4.IndexReport{Success: true},
		},
	}

	srv := NewServer()
	srv.SetReport(report)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go srv.Serve(ctx, ln)

	conn, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)

	data, err := io.ReadAll(conn)
	require.NoError(t, err)

	got := &v1.VMReport{}
	require.NoError(t, proto.Unmarshal(data, got))
	assert.Equal(t, "7", got.GetIndexReport().GetVsockCid())

	cancel()
}
