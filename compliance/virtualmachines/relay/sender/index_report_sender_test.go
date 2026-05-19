package sender

import (
	"context"
	"testing"
	"time"

	relaytest "github.com/stackrox/rox/compliance/virtualmachines/relay/testutils"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/suite"
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

func (s *senderTestSuite) TestSend_HandlesContextCancellation() {
	client := relaytest.NewMockSensorClient(s.T()).WithDelay(200 * time.Millisecond)
	sender := New(client)
	ctx, cancel := context.WithCancel(s.ctx)
	cancel()

	err := sender.Send(ctx, relaytest.NewTestVMReport("42"))
	s.Require().Error(err)
	s.Contains(err.Error(), "context canceled")
}

func (s *senderTestSuite) TestSend_ErrorBehavior() {
	cases := map[string]struct {
		err         error
		respSuccess bool
	}{
		"retryable error returns error": {
			err:         status.Error(codes.ResourceExhausted, "retryable error"),
			respSuccess: false,
		},
		"non-retryable error returns error": {
			err:         errox.NotImplemented,
			respSuccess: false,
		},
		"unsuccessful request returns error": {
			err:         nil,
			respSuccess: false,
		},
	}
	for name, c := range cases {
		s.Run(name, func() {
			client := relaytest.NewMockSensorClient(s.T()).WithError(c.err)
			if !c.respSuccess {
				client = client.WithUnsuccessfulResponse()
			}
			sender := New(client)

			// Sender performs a single RPC attempt and returns any resulting error immediately.
			ctx, cancel := context.WithTimeout(s.ctx, 500*time.Millisecond)
			defer cancel()

			err := sender.Send(ctx, relaytest.NewTestVMReport("42"))
			s.Require().Error(err)
			s.Len(client.CapturedRequests(), 1)
		})
	}
}

func (s *senderTestSuite) TestReportSender_Send() {
	client := relaytest.NewMockSensorClient(s.T())
	sender := New(client)

	msg := &v1.VMReport{
		IndexReport: &v1.IndexReport{VsockCid: "42"},
		DiscoveredData: &v1.DiscoveredData{
			DetectedOs:        v1.DetectedOS_RHEL,
			OsVersion:         "9.2",
			ActivationStatus:  v1.ActivationStatus_ACTIVE,
			DnfMetadataStatus: v1.DnfMetadataStatus_AVAILABLE,
		},
	}

	err := sender.Send(s.ctx, msg)
	s.Require().NoError(err)
	s.Require().Len(client.CapturedRequests(), 1)

	// Verify that discovered data fields are correctly forwarded into the request
	req := client.CapturedRequests()[0]
	s.Equal(msg.GetDiscoveredData().GetDetectedOs(), req.GetDiscoveredData().GetDetectedOs())
	s.Equal(msg.GetDiscoveredData().GetActivationStatus(), req.GetDiscoveredData().GetActivationStatus())
	s.Equal(msg.GetDiscoveredData().GetDnfMetadataStatus(), req.GetDiscoveredData().GetDnfMetadataStatus())
}

func (s *senderTestSuite) TestReportSender_SendHandlesErrors() {
	client := relaytest.NewMockSensorClient(s.T()).WithError(errox.NotImplemented)
	sender := New(client)

	err := sender.Send(s.ctx, relaytest.NewTestVMReport("42"))
	s.Require().Error(err)
}

func (s *senderTestSuite) TestReportSender_SendMissingIndexReport() {
	client := relaytest.NewMockSensorClient(s.T())
	sender := New(client)

	msg := &v1.VMReport{
		DiscoveredData: &v1.DiscoveredData{
			DetectedOs:        v1.DetectedOS_RHEL,
			OsVersion:         "9.2",
			ActivationStatus:  v1.ActivationStatus_ACTIVE,
			DnfMetadataStatus: v1.DnfMetadataStatus_AVAILABLE,
		},
	}

	err := sender.Send(s.ctx, msg)
	s.Require().Error(err)
	s.Require().Empty(client.CapturedRequests())
}
