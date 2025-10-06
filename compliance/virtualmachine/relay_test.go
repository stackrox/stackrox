package virtualmachine

import (
	"context"
	"net"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestVMRelay(t *testing.T) {
	suite.Run(t, new(relayTestSuite))
}

type relayTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (s *relayTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *relayTestSuite) TestExtractVsockCIDFromConnection() {

	connWrongAddrType := s.defaultVsockConn()
	connWrongAddrType.remoteAddr = &net.TCPAddr{}

	cases := map[string]struct {
		conn             net.Conn
		shouldError      bool
		expectedVsockCID uint32
	}{
		"wrong type fails": {
			conn:             connWrongAddrType,
			shouldError:      true,
			expectedVsockCID: 0,
		},
		"reserved vsock CID fails": {
			conn:             s.defaultVsockConn().withVsockCID(2),
			shouldError:      true,
			expectedVsockCID: 0,
		},
		"valid vsock CID succeeds": {
			conn:             s.defaultVsockConn().withVsockCID(42),
			shouldError:      false,
			expectedVsockCID: 42,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			vsockCID, err := extractVsockCIDFromConnection(c.conn)
			if c.shouldError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Equal(c.expectedVsockCID, vsockCID)
			}
		})
	}
}

func (s *relayTestSuite) TestHandleVsockConnection_InjectsVsockCID() {
	conn := s.defaultVsockConn().withVsockCID(42)
	client := newMockSensorClient()

	err := handleVsockConnection(s.ctx, conn, client, 10*time.Second)
	s.Require().NoError(err)

	s.Equal("42", client.capturedRequests[0].IndexReport.VsockCid)

}

func (s *relayTestSuite) TestHandleVsockConnection_RejectsMalformedData() {
	conn := s.defaultVsockConn().withData([]byte("malformed-data"))
	client := newMockSensorClient()

	err := handleVsockConnection(s.ctx, conn, client, 10*time.Second)
	s.Error(err)
}

func (s *relayTestSuite) TestHandleVsockConnection_HandlesContextCancellation() {
	conn := s.defaultVsockConn()
	client := newMockSensorClient().withDelay(1 * time.Second)
	ctx, cancel := context.WithTimeout(s.ctx, 100*time.Millisecond) // times out before sensor replies
	defer cancel()

	err := handleVsockConnection(ctx, conn, client, 10*time.Second)
	s.Require().Error(err)
	s.Contains(err.Error(), "context deadline exceeded")
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
			conn := s.defaultVsockConn().withData(data).withDelay(c.delay)

			readData, err := readFromConn(conn, c.maxSize, c.readTimeout)
			if c.shouldError {
				s.Error(err)
			} else {
				s.Require().NoError(err)
				s.Equal(data, readData)
			}
		})
	}
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
			client := newMockSensorClient().withError(c.err)
			if !c.respSuccess {
				client = client.withUnsuccessfulResponse()
			}

			// The retry logic uses withExponentialBackoff, which currently has an initial delay between retries of
			// 100 ms, therefore after 500 ms the failing call has been retried already
			ctx, cancel := context.WithTimeout(s.ctx, 500*time.Millisecond)
			defer cancel()

			err := sendReportToSensor(ctx, &v1.IndexReport{}, client)
			s.Require().Error(err)

			retried := len(client.capturedRequests) > 1
			s.Equal(c.shouldRetry, retried)
		})
	}
}

func (s *relayTestSuite) defaultVsockConn() *mockVsockConn {
	c := newMockVsockConn().withVsockCID(1234)
	c, err := c.withIndexReport(&v1.IndexReport{})
	s.Require().NoError(err)
	return c
}
