package virtualmachine

import (
	"context"
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

func (s *relayTestSuite) TestHandleVsockConnection_InjectsVsockCID() {
	conn := s.defaultVsockConn().withVsockCID(42)
	client := newMockSensorClient()

	err := handleVsockConnection(s.ctx, conn, client)
	s.Require().NoError(err)

	s.Equal("42", client.capturedRequests[0].IndexReport.VsockCid)

}

func (s *relayTestSuite) TestHandleVsockConnection_RejectsMalformedData() {
	conn := s.defaultVsockConn().withData([]byte("malformed-data"))
	client := newMockSensorClient()

	err := handleVsockConnection(s.ctx, conn, client)
	s.Error(err)
}

func (s *relayTestSuite) TestHandleVsockConnection_HandlesContextCancellation() {
	conn := s.defaultVsockConn()
	client := newMockSensorClient().withDelay(1 * time.Second)
	ctx, cancel := context.WithTimeout(s.ctx, 100*time.Millisecond) // times out before sensor replies
	defer cancel()

	err := handleVsockConnection(ctx, conn, client)
	s.Require().Error(err)
	s.Contains(err.Error(), "context deadline exceeded")
}

func (s *relayTestSuite) TestReadFromConn_EnforcesSizeLimit() {
	data := []byte("Hello, world!")

	cases := map[string]struct {
		sizeLimit   int
		shouldError bool
	}{
		"data smaller than limit succeeds": {
			sizeLimit:   2 * len(data),
			shouldError: false,
		},
		"data of equal size as limit succeeds": {
			sizeLimit:   len(data),
			shouldError: false,
		},
		"data larger than limit fails": {
			sizeLimit:   len(data) - 1,
			shouldError: true,
		},
	}

	conn := s.defaultVsockConn().withData(data)
	connTimeout := 10 * time.Second // Not relevant in these tests

	for name, c := range cases {
		s.Run(name, func() {
			readData, err := readFromConn(conn, c.sizeLimit, connTimeout)
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
