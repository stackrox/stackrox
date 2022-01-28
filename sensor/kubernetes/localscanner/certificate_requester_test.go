package localscanner

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/suite"
)

func TestCertificateRequester(t *testing.T) {
	suite.Run(t, new(certificateRequesterSuite))
}

type certificateRequesterSuite struct {
	suite.Suite
	sendC     chan *central.MsgFromSensor
	receiveC  chan *central.IssueLocalScannerCertsResponse
	requester CertificateRequester
}

func (s *certificateRequesterSuite) SetupTest() {
	s.sendC = make(chan *central.MsgFromSensor)
	s.receiveC = make(chan *central.IssueLocalScannerCertsResponse)
	s.requester = NewCertificateRequester(s.sendC, s.receiveC)
	s.requester.Start()
}

func (s *certificateRequesterSuite) TearDownTest() {
	s.requester.Stop()
}

func (s *certificateRequesterSuite) TestRequestCancellation() {
	requestCtx, cancelRequestCtx := context.WithCancel(context.Background())
	doneErrSig := concurrency.NewErrorSignal()

	go func() {
		certs, err := s.requester.RequestCertificates(requestCtx)
		s.Nil(certs)
		doneErrSig.SignalWithError(err)
	}()
	cancelRequestCtx()

	requestErr, ok := doneErrSig.WaitWithTimeout(100 * time.Millisecond)
	s.Require().True(ok)
	s.Equal(context.Canceled, requestErr)
}

func (s *certificateRequesterSuite) TestRequestSuccess() {
	waitCtx, cancelWaitCtx := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancelWaitCtx()

	var interceptedRequestID string
	go func() {
		select {
		case <-waitCtx.Done():
			return
		case request := <-s.sendC:
			interceptedRequestID = request.GetIssueLocalScannerCertsRequest().GetRequestId()
			s.NotEmpty(interceptedRequestID)
			s.receiveC <- &central.IssueLocalScannerCertsResponse{
				RequestId: interceptedRequestID,
			}
		}
	}()

	response, err := s.requester.RequestCertificates(waitCtx)
	s.NoError(err)
	s.Equal(interceptedRequestID, response.GetRequestId())
}

func (s *certificateRequesterSuite) TestResponsesWithUnknownIDAreIgnored() {
	waitCtx, cancelWaitCtx := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancelWaitCtx()
	doneErrSig := concurrency.NewErrorSignal()

	go func() {
		select {
		case <-waitCtx.Done():
		case <-s.sendC:
			select {
			case <-waitCtx.Done():
				// Request with different request ID should be ignored.
			case s.receiveC <- &central.IssueLocalScannerCertsResponse{RequestId: ""}:
			}
		}
	}()

	go func() {
		certs, err := s.requester.RequestCertificates(waitCtx)
		s.Nil(certs)
		doneErrSig.SignalWithError(err)
	}()

	requestErr, ok := doneErrSig.WaitWithTimeout(100 * time.Millisecond)
	s.Require().True(ok)
	s.Equal(context.DeadlineExceeded, requestErr)
}

func (s *certificateRequesterSuite) TestRequestConcurrentRequestDoNotInterfere() {
	waitCtx, cancelWaitCtx := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelWaitCtx()
	numConcurrentRequests := 2
	waitGroup := concurrency.NewWaitGroup(numConcurrentRequests)

	for i := 0; i < numConcurrentRequests; i++ {
		go func() {
			select {
			case <-waitCtx.Done():
				return
			case request := <-s.sendC:
				interceptedRequestID := request.GetIssueLocalScannerCertsRequest().GetRequestId()
				s.NotEmpty(interceptedRequestID)
				s.receiveC <- &central.IssueLocalScannerCertsResponse{
					RequestId: interceptedRequestID,
				}
			}
		}()

		go func() {
			_, err := s.requester.RequestCertificates(waitCtx)
			s.NoError(err)
			waitGroup.Add(-1)
		}()
	}

	ok := concurrency.WaitWithTimeout(&waitGroup, 2*time.Second)
	s.Require().True(ok)
}
