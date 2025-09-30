package virtualmachine

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/mdlayher/vsock"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type mockVsockConn struct {
	closed     bool
	data       []byte
	readErr    error
	remoteAddr *vsock.Addr
}

func newMockVsockConn() *mockVsockConn {
	return &mockVsockConn{}
}

func (c *mockVsockConn) withVsockCID(vsockCID uint32) *mockVsockConn {
	c.remoteAddr = &vsock.Addr{ContextID: vsockCID}
	return c
}

func (c *mockVsockConn) withIndexReport(indexReport *v1.IndexReport) (*mockVsockConn, error) {
	reportData, err := proto.Marshal(indexReport)
	c.data = reportData
	return c, err
}

func (c *mockVsockConn) withData(data []byte) *mockVsockConn {
	c.data = data
	return c
}

func (c *mockVsockConn) Read(b []byte) (n int, err error) {
	if c.readErr != nil {
		return 0, c.readErr
	}
	n = copy(b, c.data)
	if n == len(c.data) {
		return n, io.EOF
	}
	return n, nil
}

func (c *mockVsockConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *mockVsockConn) Close() error {
	c.closed = true
	return nil
}

func (c *mockVsockConn) Write([]byte) (int, error)        { return 0, nil }
func (c *mockVsockConn) LocalAddr() net.Addr              { return nil }
func (c *mockVsockConn) SetDeadline(time.Time) error      { return nil }
func (c *mockVsockConn) SetReadDeadline(time.Time) error  { return nil }
func (c *mockVsockConn) SetWriteDeadline(time.Time) error { return nil }

type mockSensorClient struct {
	capturedRequests []*sensor.UpsertVirtualMachineIndexReportRequest
	delay            time.Duration
	err              error
	response         *sensor.UpsertVirtualMachineIndexReportResponse
}

func newMockSensorClient() *mockSensorClient {
	return &mockSensorClient{
		response: &sensor.UpsertVirtualMachineIndexReportResponse{Success: true},
	}
}

func (c *mockSensorClient) UpsertVirtualMachineIndexReport(ctx context.Context, req *sensor.UpsertVirtualMachineIndexReportRequest, _ ...grpc.CallOption) (*sensor.UpsertVirtualMachineIndexReportResponse, error) {
	select {
	case <-time.After(c.delay):
		c.capturedRequests = append(c.capturedRequests, req)
		return c.response, c.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *mockSensorClient) withDelay(delay time.Duration) *mockSensorClient {
	c.delay = delay
	return c
}

func (c *mockSensorClient) withError(err error) *mockSensorClient {
	c.err = err
	return c
}

func (c *mockSensorClient) withUnsuccessfulResponse() *mockSensorClient {
	c.response = &sensor.UpsertVirtualMachineIndexReportResponse{Success: false}
	return c
}
