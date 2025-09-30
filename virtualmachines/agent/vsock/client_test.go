package vsock

import (
	"fmt"
	"net"
	"testing"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func setupLocalTCPListener(t *testing.T) (net.Listener, uint64) {
	port := testutils.GetFreeTestPort()
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err, "failed to create local TCP listener")

	t.Cleanup(func() {
		listener.Close()
	})

	return listener, port
}

func TestClient_writeIndexReport_LocalSocket(t *testing.T) {
	listener, port := setupLocalTCPListener(t)
	client := &Client{Port: uint32(port)}
	testReport := &v4.IndexReport{
		HashId:  "test-hash-local",
		State:   "completed",
		Success: true,
	}

	receivedData := make(chan []byte, 1)
	listenerErr := make(chan error, 1)

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			listenerErr <- fmt.Errorf("accept failed: %w", err)
			return
		}
		defer conn.Close()

		buf := make([]byte, 4096)
		n, err := conn.Read(buf)
		if err != nil {
			listenerErr <- fmt.Errorf("read failed: %w", err)
			return
		}

		receivedData <- buf[:n]
	}()

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err, "should be able to dial local TCP socket")
	defer conn.Close()

	err = client.writeIndexReport(conn, testReport)
	require.NoError(t, err, "writeIndexReport should succeed")

	select {
	case data := <-receivedData:
		var receivedReport v4.IndexReport
		err = proto.Unmarshal(data, &receivedReport)
		require.NoError(t, err, "should be able to unmarshal received data")
		protoassert.Equal(t, testReport, &receivedReport)
	case err := <-listenerErr:
		close(receivedData)
		t.Fatalf("listener error: %v", err)
	case <-time.After(5 * time.Second):
		close(receivedData)
		t.Fatal("timeout waiting for data")
	}
}
