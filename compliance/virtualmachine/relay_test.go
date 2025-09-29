package virtualmachine

import (
	"context"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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

func (s *relayTestSuite) TestVsockConnectionHandlerInjectsVsockCID() {
	conn := s.defaultVsockConn().withVsockCID(42)
	client := newMockSensorClient()

	err := handleVsockConnection(s.ctx, conn, client)
	require.NoError(s.T(), err)

	s.True(conn.closed, "connection should be closed after handling")

	s.Equal("42", client.capturedRequests[0].IndexReport.VsockCid)
}

func (s *relayTestSuite) TestVsockConnectionHandlerRejectsMalformedData() {
	conn := s.defaultVsockConn().withData([]byte("malformed-data"))
	client := newMockSensorClient()

	err := handleVsockConnection(s.ctx, conn, client)
	require.Error(s.T(), err)

	s.True(conn.closed, "connection should be closed after handling")
}

func (s *relayTestSuite) TestReadFromConn() {
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

	for name, c := range cases {
		s.Run(name, func() {
			readData, err := readFromConn(conn, c.sizeLimit)
			if c.shouldError {
				s.Error(err)
			} else {
				s.NoError(err)
				s.Equal(data, readData)
			}
		})
	}
}

func (s *relayTestSuite) TestSendReportToSensorRetries() {
	conn := s.defaultVsockConn()

	cases := map[string]struct {
		err         error
		respSuccess bool
		shouldRetry bool
	}{
		"retryable error is retried": {
			err:         errox.ResourceExhausted,
			respSuccess: false,
			shouldRetry: true,
		},
		"nonretryable error is not retried": {
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

			err := handleVsockConnection(ctx, conn, client)

			s.Error(err)
			s.True(conn.closed, "connection should be closed after handling")

			retried := len(client.capturedRequests) > 1

			s.Equal(c.shouldRetry, retried)
		})
	}
}

func (s *relayTestSuite) defaultVsockConn() *mockVsockConn {
	c := newMockVsockConn().withVsockCID(1234)
	c, err := c.withIndexReport(&v1.IndexReport{})
	s.NoError(err)
	return c
}
