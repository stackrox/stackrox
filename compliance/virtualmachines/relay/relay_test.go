package relay

//go:generate mockgen-wrapper Conn net
//go:generate mockgen-wrapper VirtualMachineIndexReportServiceClient github.com/stackrox/rox/generated/internalapi/sensor

import (
	"context"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/mdlayher/vsock"
	sensormocks "github.com/stackrox/rox/compliance/virtualmachines/relay/mocks/github.com/stackrox/rox/generated/internalapi/sensor/mocks"
	netmocks "github.com/stackrox/rox/compliance/virtualmachines/relay/mocks/net/mocks"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestVMRelay(t *testing.T) {
	suite.Run(t, new(relayTestSuite))
}

type relayTestSuite struct {
	suite.Suite

	ctx      context.Context
	mockCtrl *gomock.Controller
}

func (s *relayTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *relayTestSuite) TearDownTest() {
	if s.mockCtrl != nil {
		s.mockCtrl.Finish()
	}
}

func (s *relayTestSuite) TestExtractVsockCIDFromConnection() {
	cases := map[string]struct {
		setupConn        func() net.Conn
		shouldError      bool
		expectedVsockCID uint32
	}{
		"wrong type fails": {
			setupConn: func() net.Conn {
				conn := netmocks.NewMockConn(s.mockCtrl)
				conn.EXPECT().RemoteAddr().Return(&net.TCPAddr{}).AnyTimes()
				return conn
			},
			shouldError:      true,
			expectedVsockCID: 0,
		},
		"reserved vsock CID fails": {
			setupConn: func() net.Conn {
				conn := netmocks.NewMockConn(s.mockCtrl)
				conn.EXPECT().RemoteAddr().Return(&vsock.Addr{ContextID: 2}).AnyTimes()
				return conn
			},
			shouldError:      true,
			expectedVsockCID: 0,
		},
		"valid vsock CID succeeds": {
			setupConn: func() net.Conn {
				conn := netmocks.NewMockConn(s.mockCtrl)
				conn.EXPECT().RemoteAddr().Return(&vsock.Addr{ContextID: 42}).AnyTimes()
				return conn
			},
			shouldError:      false,
			expectedVsockCID: 42,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			conn := c.setupConn()
			vsockCID, err := extractVsockCIDFromConnection(conn)
			if c.shouldError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Equal(c.expectedVsockCID, vsockCID)
			}
		})
	}
}

func (s *relayTestSuite) TestParseIndexReport() {
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

func (s *relayTestSuite) TestReadFromConn() {
	data := []byte("Hello, world!")

	cases := map[string]struct {
		delay       time.Duration
		maxSize     int
		readTimeout time.Duration
		shouldError bool
	}{
		"data smaller than limit succeeds": {
			maxSize:     2 * len(data),
			readTimeout: 10 * time.Second,
			shouldError: false,
		},
		"data of equal size as limit succeeds": {
			maxSize:     len(data),
			readTimeout: 10 * time.Second,
			shouldError: false,
		},
		"data larger than limit fails": {
			maxSize:     len(data) - 1,
			readTimeout: 10 * time.Second,
			shouldError: true,
		},
		"delay longer than timeout fails": {
			maxSize:     len(data),
			delay:       1 * time.Second,
			readTimeout: 100 * time.Millisecond,
			shouldError: true,
		},
		"delay shorter than timeout succeeds": {
			maxSize:     len(data),
			delay:       100 * time.Millisecond,
			readTimeout: 1 * time.Second,
			shouldError: false,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			conn := netmocks.NewMockConn(s.mockCtrl)

			var readDeadline time.Time
			conn.EXPECT().SetReadDeadline(gomock.Any()).DoAndReturn(func(t time.Time) error {
				readDeadline = t
				return nil
			})

			conn.EXPECT().Read(gomock.Any()).DoAndReturn(func(b []byte) (int, error) {
				time.Sleep(c.delay)
				if !readDeadline.IsZero() && time.Now().After(readDeadline) {
					return 0, os.ErrDeadlineExceeded
				}
				n := copy(b, data)
				if n == len(data) {
					return n, io.EOF
				}
				return n, nil
			}).AnyTimes()

			readData, err := readFromConn(conn, c.maxSize, c.readTimeout, 12345)
			if c.shouldError {
				s.Error(err)
			} else {
				s.Require().NoError(err)
				s.Equal(data, readData)
			}
		})
	}
}

func (s *relayTestSuite) TestSemaphore() {
	vsockServer := &vsockServerImpl{
		semaphore:        semaphore.NewWeighted(1),
		semaphoreTimeout: 5 * time.Millisecond,
	}

	// First should succeed
	err := vsockServer.acquireSemaphore(s.ctx)
	s.Require().NoError(err)

	// Second should time out
	err = vsockServer.acquireSemaphore(s.ctx)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "failed to acquire semaphore")

	// After releasing once, a new acquire should succeed
	vsockServer.releaseSemaphore()
	err = vsockServer.acquireSemaphore(s.ctx)
	s.Require().NoError(err)
}

func (s *relayTestSuite) TestSendReportToSensor_HandlesContextCancellation() {
	client := sensormocks.NewMockVirtualMachineIndexReportServiceClient(s.mockCtrl)
	ctx, cancel := context.WithCancel(s.ctx)

	client.EXPECT().UpsertVirtualMachineIndexReport(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, req *sensor.UpsertVirtualMachineIndexReportRequest, opts ...interface{}) (*sensor.UpsertVirtualMachineIndexReportResponse, error) {
			select {
			case <-time.After(500 * time.Millisecond):
				return &sensor.UpsertVirtualMachineIndexReportResponse{Success: true}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}).AnyTimes()

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := sendReportToSensor(ctx, &v1.IndexReport{}, client)
	s.Require().Error(err)
	s.Contains(err.Error(), "context canceled")
}

func (s *relayTestSuite) TestSendReportToSensor_RetriesOnRetryableErrors() {
	cases := map[string]struct {
		err         error
		respSuccess bool
		shouldRetry bool
	}{
		"retryable error is retried": {
			err:         status.Error(codes.ResourceExhausted, "retryable error"),
			respSuccess: false,
			shouldRetry: true,
		},
		"non-retryable error is not retried": {
			err:         errox.NotImplemented,
			respSuccess: false,
			shouldRetry: false,
		},
		"Unsuccessful request is retried": {
			err:         nil,
			respSuccess: false,
			shouldRetry: true,
		},
	}
	for name, c := range cases {
		s.Run(name, func() {
			client := sensormocks.NewMockVirtualMachineIndexReportServiceClient(s.mockCtrl)

			var callCount int
			client.EXPECT().UpsertVirtualMachineIndexReport(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, req *sensor.UpsertVirtualMachineIndexReportRequest, opts ...interface{}) (*sensor.UpsertVirtualMachineIndexReportResponse, error) {
					callCount++
					return &sensor.UpsertVirtualMachineIndexReportResponse{Success: c.respSuccess}, c.err
				}).AnyTimes()

			// The retry logic uses withExponentialBackoff, which currently has an initial delay between retries of
			// 100 ms, therefore after 500 ms the failing call has been retried already
			ctx, cancel := context.WithTimeout(s.ctx, 500*time.Millisecond)
			defer cancel()

			err := sendReportToSensor(ctx, &v1.IndexReport{}, client)
			s.Require().Error(err)

			retried := callCount > 1
			s.Equal(c.shouldRetry, retried)
		})
	}
}

func (s *relayTestSuite) TestValidateVsockCID() {
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
