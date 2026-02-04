package relaytest

import (
	"context"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/mdlayher/vsock"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// MockVsockConn implements net.Conn and allows tests to craft vsock-backed connections easily.
type MockVsockConn struct {
	closed       bool
	data         []byte
	delay        time.Duration
	readErr      error
	remoteAddr   net.Addr
	readDeadline time.Time
}

func NewMockVsockConn(_ testing.TB) *MockVsockConn {
	return &MockVsockConn{}
}

func (c *MockVsockConn) WithVsockCID(vsockCID uint32) *MockVsockConn {
	c.remoteAddr = &vsock.Addr{ContextID: vsockCID}
	return c
}

func (c *MockVsockConn) WithData(data []byte) *MockVsockConn {
	c.data = data
	return c
}

func (c *MockVsockConn) WithVMReport(vmReport *v1.VMReport) (*MockVsockConn, error) {
	data, err := proto.Marshal(vmReport)
	if err != nil {
		return nil, err
	}
	c.data = data
	return c, nil
}

// Deprecated: Use WithVMReport instead
func (c *MockVsockConn) WithIndexReport(indexReport *v1.IndexReport) (*MockVsockConn, error) {
	vmReport := &v1.VMReport{
		IndexReport: indexReport,
	}
	return c.WithVMReport(vmReport)
}

// NewTestVMReport creates a VMReport with default test values.
func NewTestVMReport(vsockCID string) *v1.VMReport {
	return &v1.VMReport{
		IndexReport: &v1.IndexReport{VsockCid: vsockCID},
		DiscoveredData: &v1.DiscoveredData{
			DetectedOs:        v1.DetectedOS_UNKNOWN,
			OsVersion:         "",
			ActivationStatus:  v1.ActivationStatus_ACTIVATION_UNSPECIFIED,
			DnfMetadataStatus: v1.DnfMetadataStatus_DNF_METADATA_UNSPECIFIED,
		},
	}
}

func (c *MockVsockConn) WithDelay(delay time.Duration) *MockVsockConn {
	c.delay = delay
	return c
}

func (c *MockVsockConn) SetRemoteAddr(addr net.Addr) {
	c.remoteAddr = addr
}

func (c *MockVsockConn) Read(b []byte) (int, error) {
	time.Sleep(c.delay)
	if !c.readDeadline.IsZero() && time.Now().After(c.readDeadline) {
		return 0, os.ErrDeadlineExceeded
	}
	if c.readErr != nil {
		return 0, c.readErr
	}
	n := copy(b, c.data)
	if n == len(c.data) {
		return n, io.EOF
	}
	return n, nil
}

func (c *MockVsockConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *MockVsockConn) Close() error {
	c.closed = true
	return nil
}

func (c *MockVsockConn) Write([]byte) (int, error)         { return 0, nil }
func (c *MockVsockConn) LocalAddr() net.Addr               { return nil }
func (c *MockVsockConn) SetDeadline(time.Time) error       { return nil }
func (c *MockVsockConn) SetReadDeadline(t time.Time) error { c.readDeadline = t; return nil }
func (c *MockVsockConn) SetWriteDeadline(time.Time) error  { return nil }

// MockSensorClient captures UpsertVirtualMachineIndexReport calls and allows injecting delays or errors.
type MockSensorClient struct {
	capturedRequests []*sensor.UpsertVirtualMachineIndexReportRequest
	delay            time.Duration
	err              error
	response         *sensor.UpsertVirtualMachineIndexReportResponse
}

func NewMockSensorClient(_ testing.TB) *MockSensorClient {
	return &MockSensorClient{
		response: &sensor.UpsertVirtualMachineIndexReportResponse{Success: true},
	}
}

func (m *MockSensorClient) WithDelay(delay time.Duration) *MockSensorClient {
	m.delay = delay
	return m
}

func (m *MockSensorClient) WithError(err error) *MockSensorClient {
	m.err = err
	return m
}

func (m *MockSensorClient) WithUnsuccessfulResponse() *MockSensorClient {
	m.response = &sensor.UpsertVirtualMachineIndexReportResponse{Success: false}
	return m
}

func (m *MockSensorClient) CapturedRequests() []*sensor.UpsertVirtualMachineIndexReportRequest {
	return m.capturedRequests
}

func (m *MockSensorClient) UpsertVirtualMachineIndexReport(ctx context.Context, req *sensor.UpsertVirtualMachineIndexReportRequest, _ ...grpc.CallOption) (*sensor.UpsertVirtualMachineIndexReportResponse, error) {
	select {
	case <-time.After(m.delay):
		m.capturedRequests = append(m.capturedRequests, req)
		return m.response, m.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
