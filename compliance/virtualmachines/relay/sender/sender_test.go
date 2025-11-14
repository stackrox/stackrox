package sender

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSender(t *testing.T) {
	suite.Run(t, new(senderTestSuite))
}

type senderTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (s *senderTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *senderTestSuite) TestSendReportToSensor_HandlesContextCancellation() {
	client := newMockSensorClient()
	ctx, cancel := context.WithCancel(s.ctx)
	cancel()

	err := SendReportToSensor(ctx, &v1.IndexReport{}, client)
	s.Require().Error(err)
	s.Contains(err.Error(), "context canceled")
}

func (s *senderTestSuite) TestSendReportToSensor_RetriesOnRetryableErrors() {
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

			err := SendReportToSensor(ctx, &v1.IndexReport{}, client)
			s.Require().Error(err)

			retried := len(client.capturedRequests) > 1
			s.Equal(c.shouldRetry, retried)
		})
	}
}

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

func (m *mockSensorClient) withError(err error) *mockSensorClient {
	m.err = err
	return m
}

func (m *mockSensorClient) withUnsuccessfulResponse() *mockSensorClient {
	m.response = &sensor.UpsertVirtualMachineIndexReportResponse{Success: false}
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
