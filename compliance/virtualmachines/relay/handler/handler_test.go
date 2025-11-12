package handler

import (
	"context"
	"io"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/mdlayher/vsock"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

func TestHandler(t *testing.T) {
	suite.Run(t, new(handlerTestSuite))
}

type handlerTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (s *handlerTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *handlerTestSuite) TestParseIndexReport() {
	data := []byte("malformed-data")
	parsedIndexReport, err := parseIndexReport(data)
	s.Require().Error(err)
	s.Require().Nil(parsedIndexReport)

	validIndexReport := &v1.IndexReport{VsockCid: "42"}
	data, err = proto.Marshal(validIndexReport)
	s.Require().NoError(err)
	parsedIndexReport, err = parseIndexReport(data)
	s.Require().NoError(err)
	s.Require().True(proto.Equal(validIndexReport, parsedIndexReport))
}

func (s *handlerTestSuite) TestValidateVsockCID() {
	// Reported CID is 42
	indexReport := v1.IndexReport{VsockCid: "42"}

	// Real (connection) CID is 99 - does not match, should return error
	connVsockCID := uint32(99)
	err := validateReportedVsockCID(&indexReport, connVsockCID)
	s.Require().Error(err)

	// Real (connection) CID is 42 - matches, should return nil
	connVsockCID = uint32(42)
	err = validateReportedVsockCID(&indexReport, connVsockCID)
	s.Require().NoError(err)
}

func (s *handlerTestSuite) TestHandle_RejectsMismatchingVsockCID() {
	cases := map[string]struct {
		indexReportVsockCID int
		connVsockCID        int
		shouldError         bool
	}{
		"mismatching vsock CID fails": {
			indexReportVsockCID: 42,
			connVsockCID:        99,
			shouldError:         true,
		},
		"matching vsock CID succeeds": {
			indexReportVsockCID: 42,
			connVsockCID:        42,
			shouldError:         false,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			indexReport := &v1.IndexReport{VsockCid: strconv.Itoa(c.indexReportVsockCID)}
			conn, err := newMockVsockConn().withVsockCID(uint32(c.connVsockCID)).withIndexReport(indexReport)
			s.Require().NoError(err)
			client := newMockSensorClient()

			handler := New(client)
			err = handler.Handle(s.ctx, conn)
			if c.shouldError {
				s.Require().Error(err)
				s.Contains(err.Error(), "mismatch")
				s.Empty(client.capturedRequests)
			} else {
				s.Require().NoError(err)
				s.Len(client.capturedRequests, 1)
			}
		})
	}
}

func (s *handlerTestSuite) TestHandle_RejectsMalformedData() {
	conn := newMockVsockConn().withVsockCID(1234).withData([]byte("malformed-data"))
	client := newMockSensorClient()
	handler := New(client)

	err := handler.Handle(s.ctx, conn)
	s.Error(err)
}

func (s *handlerTestSuite) TestHandle_HandlesContextCancellation() {
	indexReport := &v1.IndexReport{VsockCid: "1234"}
	conn, err := newMockVsockConn().withVsockCID(1234).withIndexReport(indexReport)
	s.Require().NoError(err)

	// Set up a sensor client that only returns after 500 ms
	client := newMockSensorClient().withDelay(500 * time.Millisecond)

	// Set up a context that will be canceled after 100 ms
	cancellableCtx, cancel := context.WithCancel(s.ctx)
	handler := New(client)
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// When the connection is handled, sending the index report to sensor will hang for 500 ms.
	// The context will be canceled after 100 ms, that is, while the sensor response is awaited.
	// Therefore we expect a "context canceled" error
	err = handler.Handle(cancellableCtx, conn)
	s.Require().Error(err)
	s.Contains(err.Error(), "context canceled")
}

// mockVsockConn for integration tests
type mockVsockConn struct {
	closed       bool
	data         []byte
	delay        time.Duration
	remoteAddr   net.Addr
	readDeadline time.Time
}

func newMockVsockConn() *mockVsockConn {
	return &mockVsockConn{}
}

func (c *mockVsockConn) withVsockCID(vsockCID uint32) *mockVsockConn {
	c.remoteAddr = &vsock.Addr{ContextID: vsockCID}
	return c
}

func (c *mockVsockConn) withData(data []byte) *mockVsockConn {
	c.data = data
	return c
}

func (c *mockVsockConn) withIndexReport(indexReport *v1.IndexReport) (*mockVsockConn, error) {
	data, err := proto.Marshal(indexReport)
	if err != nil {
		return nil, err
	}
	c.data = data
	return c, nil
}

func (c *mockVsockConn) Read(b []byte) (n int, err error) {
	time.Sleep(c.delay)
	if !c.readDeadline.IsZero() && time.Now().After(c.readDeadline) {
		return 0, os.ErrDeadlineExceeded
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

func (c *mockVsockConn) Write([]byte) (int, error)         { return 0, nil }
func (c *mockVsockConn) LocalAddr() net.Addr               { return nil }
func (c *mockVsockConn) SetDeadline(time.Time) error       { return nil }
func (c *mockVsockConn) SetReadDeadline(t time.Time) error { c.readDeadline = t; return nil }
func (c *mockVsockConn) SetWriteDeadline(time.Time) error  { return nil }

// mockSensorClient for integration tests
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

func (m *mockSensorClient) withDelay(delay time.Duration) *mockSensorClient {
	m.delay = delay
	return m
}

func (m *mockSensorClient) UpsertVirtualMachineIndexReport(ctx context.Context, req *sensor.UpsertVirtualMachineIndexReportRequest, _ ...grpc.CallOption) (*sensor.UpsertVirtualMachineIndexReportResponse, error) {
	select {
	case <-time.After(m.delay):
		m.capturedRequests = append(m.capturedRequests, req)
		return m.response, m.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
