package localscanner

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/suite"
)

const (
	numConcurrentRequests = 2
)

var (
	testTimeout = time.Second
)

func TestCertificateRequester(t *testing.T) {
	suite.Run(t, new(certificateRequesterSuite))
}

type certificateRequesterSuite struct {
	suite.Suite
	sendC                chan *central.MsgFromSensor
	receiveC             chan *central.IssueLocalScannerCertsResponse
	requester            CertificateRequester
	interceptedRequestID atomic.Value
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

	requestErr, ok := doneErrSig.WaitWithTimeout(testTimeout)
	s.Require().True(ok)
	s.Equal(context.Canceled, requestErr)
}

func (s *certificateRequesterSuite) TestRequestSuccess() {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	go s.respondRequest(ctx, nil)

	response, err := s.requester.RequestCertificates(ctx)
	s.NoError(err)
	s.Equal(s.interceptedRequestID.Load(), response.GetRequestId())
}

func (s *certificateRequesterSuite) TestResponsesWithUnknownIDAreIgnored() {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Request with different request ID should be ignored.
	go s.respondRequest(ctx, &central.IssueLocalScannerCertsResponse{RequestId: "UNKNOWN"})

	certs, requestErr := s.requester.RequestCertificates(ctx)
	s.Nil(certs)
	s.Equal(context.DeadlineExceeded, requestErr)
}

func (s *certificateRequesterSuite) TestRequestConcurrentRequestDoNotInterfere() {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	waitGroup := concurrency.NewWaitGroup(numConcurrentRequests)

	for i := 0; i < numConcurrentRequests; i++ {
		go s.respondRequest(ctx, nil)

		go func() {
			defer waitGroup.Add(-1)
			_, err := s.requester.RequestCertificates(ctx)
			s.NoError(err)
		}()
	}

	ok := concurrency.WaitWithTimeout(&waitGroup, time.Duration(numConcurrentRequests)*testTimeout)
	s.Require().True(ok)
}

// respondRequest reads a request from `s.sendC` and responds with `responseRequestID` as the requestID, or with
// the same ID as the request if `responseRequestID` is "".
// Before sending the response, it stores in s.responseRequestID the request ID for the requests read from `s.sendC`.
func (s *certificateRequesterSuite) respondRequest(ctx context.Context, responseOverwrite *central.IssueLocalScannerCertsResponse) {
	select {
	case <-ctx.Done():
	case request := <-s.sendC:
		interceptedRequestID := request.GetIssueLocalScannerCertsRequest().GetRequestId()
		s.NotEmpty(interceptedRequestID)
		var response *central.IssueLocalScannerCertsResponse
		if responseOverwrite != nil {
			response = responseOverwrite
		} else {
			response = &central.IssueLocalScannerCertsResponse{RequestId: interceptedRequestID}
		}
		s.interceptedRequestID.Store(response.GetRequestId())
		select {
		case <-ctx.Done():
		case s.receiveC <- response:
		}
	}
}
