package localscanner

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/suite"
)

func TestHandler(t *testing.T) {
	suite.Run(t, new(certificateRequesterSuite))
}

type certificateRequesterSuite struct {
	suite.Suite
}

type fixture struct {
	msgFromSensorC chan *central.MsgFromSensor
	msgToSensorC   chan *central.IssueLocalScannerCertsResponse
	requester      CertificateRequester
}

func newFixture() *fixture {
	msgFromSensorC := make(chan *central.MsgFromSensor)
	msgToSensorC := make(chan *central.IssueLocalScannerCertsResponse)
	return &fixture{
		msgFromSensorC: msgFromSensorC,
		msgToSensorC: msgToSensorC,
		requester: NewCertificateRequester(msgFromSensorC, msgToSensorC),
	}
}

func (s *certificateRequesterSuite) TestRequestCancellation() {
	f := newFixture()
	requestCtx, cancelRequestCtx := context.WithCancel(context.Background())
	doneErrSig := concurrency.NewErrorSignal()

	go func() {
		certs, err := f.requester.RequestCertificates(requestCtx)
		s.Nil(certs)
		doneErrSig.SignalWithError(err)
	}()
	cancelRequestCtx()

	waitCtx, cancelWaitCtx := context.WithTimeout(context.Background(), time.Second)
	defer cancelWaitCtx()
	requestErr, ok := doneErrSig.WaitUntil(waitCtx)
	s.Require().True(ok)
	s.Equal(context.Canceled, requestErr)
}

func (s *certificateRequesterSuite) TestRequestSuccess() {
	f := newFixture()
	waitCtx, cancelWaitCtx := context.WithTimeout(context.Background(), time.Second)
	defer cancelWaitCtx()
	doneErrSig := concurrency.NewErrorSignal()
	expectedResponseC := make(chan *central.IssueLocalScannerCertsResponse)

	go func() {
		response, err := f.requester.RequestCertificates(waitCtx)
		expectedResponse := <-expectedResponseC
		s.Equal(expectedResponse, response)
		s.Nil(err)
		doneErrSig.Signal()
	}()

	go func() {
		select {
		case <-waitCtx.Done():
			return
		case request := <-f.msgFromSensorC:
			s.Require().NotNil(request.GetIssueLocalScannerCertsRequest())
			requestID := request.GetIssueLocalScannerCertsRequest().GetRequestId()
			s.Require().NotEmpty(requestID)
			// should be ignored.
			f.msgToSensorC <-&central.IssueLocalScannerCertsResponse{
				RequestId: "",
			}
			expectedResponse := &central.IssueLocalScannerCertsResponse{
				RequestId: requestID,
			}
			f.msgToSensorC <-expectedResponse
			expectedResponseC <-expectedResponse
		}
	}()

	_, ok := doneErrSig.WaitUntil(waitCtx)
	s.Require().True(ok)
}
