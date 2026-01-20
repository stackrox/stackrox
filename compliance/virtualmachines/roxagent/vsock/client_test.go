package vsock

import (
	"fmt"
	"net"
	"testing"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func setupLocalTCPListener(t *testing.T) (net.Listener, uint64) {
	port := testutils.GetFreeTestPort()
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err, "failed to create local TCP listener")

	t.Cleanup(func() {
		utils.IgnoreError(listener.Close)
	})

	return listener, port
}

func TestClient_writeVMReport_LocalSocket(t *testing.T) {
	listener, port := setupLocalTCPListener(t)
	client := &Client{Port: uint32(port)}
	testIndexReport := &v1.IndexReport{
		IndexV4: &v4.IndexReport{
			HashId:  "test-hash-local",
			State:   "completed",
			Success: true,
		},
	}
	testVMReport := &v1.VMReport{
		IndexReport: testIndexReport,
		DiscoveredData: &v1.DiscoveredData{
			DetectedOs:        v1.DetectedOS_UNKNOWN,
			OsVersion:         "",
			ActivationStatus:  v1.ActivationStatus_ACTIVATION_UNSPECIFIED,
			DnfMetadataStatus: v1.DnfMetadataStatus_DNF_METADATA_UNSPECIFIED,
		},
	}

	receivedData := make(chan []byte, 1)
	listenerErr := make(chan error, 1)
	defer func() {
		close(receivedData)
		close(listenerErr)
	}()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			listenerErr <- fmt.Errorf("accept failed: %w", err)
			return
		}
		defer utils.IgnoreError(listener.Close)

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
	defer utils.IgnoreError(listener.Close)

	err = client.writeVMReport(conn, testVMReport)
	require.NoError(t, err, "writeVMReport should succeed")

	select {
	case data := <-receivedData:
		var receivedReport v1.VMReport
		err = proto.Unmarshal(data, &receivedReport)
		require.NoError(t, err, "should be able to unmarshal received data")
		protoassert.Equal(t, testVMReport, &receivedReport)
	case err := <-listenerErr:
		require.NoError(t, err, "listener error")
	case <-time.After(5 * time.Second):
		t.Errorf("timeout waiting for data")
	}
}
