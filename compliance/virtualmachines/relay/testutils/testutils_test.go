package relaytest

import (
	"context"
	"testing"
	"time"

	"github.com/mdlayher/vsock"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stretchr/testify/require"
)

func TestMockVsockConn(t *testing.T) {
	conn := NewMockVsockConn().
		WithVsockCID(1234).
		WithData([]byte("hello")).
		WithDelay(10 * time.Millisecond)

	require.Equal(t, &vsock.Addr{ContextID: 1234}, conn.RemoteAddr())

	buf := make([]byte, 5)
	n, err := conn.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 5, n)
	require.Equal(t, []byte("hello"), buf)
}

func TestMockSensorClient(t *testing.T) {
	client := NewMockSensorClient()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := &sensor.UpsertVirtualMachineIndexReportRequest{
		IndexReport: &v1.IndexReport{VsockCid: "1"},
	}

	// Ensure first request is captured.
	resp, err := client.UpsertVirtualMachineIndexReport(ctx, req)
	require.NoError(t, err)
	require.True(t, resp.GetSuccess())
	require.Len(t, client.CapturedRequests(), 1)
}
