package localscanner

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	numConcurrentRequests = 2
)

var (
	testTimeout = time.Second
)

func TestCertificateRequesterRequestFailureIfStopped(t *testing.T) {
	f := newFixture(0)
	defer f.tearDown()
	doneErrSig := concurrency.NewErrorSignal()

	f.requester.Stop()
	go func() {
		certs, err := f.requester.RequestCertificates(f.ctx)
		assert.Nil(t, certs)
		doneErrSig.SignalWithError(err)
	}()

	requestErr, ok := doneErrSig.WaitWithTimeout(testTimeout)
	require.True(t, ok)
	assert.Equal(t, ErrCertificateRequesterStopped, requestErr)
}

func TestCertificateRequesterRequestCancellation(t *testing.T) {
	f := newFixture(0)
	defer f.tearDown()
	doneErrSig := concurrency.NewErrorSignal()

	go func() {
		certs, err := f.requester.RequestCertificates(f.ctx)
		assert.Nil(t, certs)
		doneErrSig.SignalWithError(err)
	}()
	f.cancelCtx()

	requestErr, ok := doneErrSig.WaitWithTimeout(testTimeout)
	require.True(t, ok)
	assert.Equal(t, context.Canceled, requestErr)
}

func TestCertificateRequesterRequestSuccess(t *testing.T) {
	f := newFixture(0)
	defer f.tearDown()

	go f.respondRequest(t, nil)

	response, err := f.requester.RequestCertificates(f.ctx)
	assert.NoError(t, err)
	assert.Equal(t, f.interceptedRequestID.Load(), response.GetRequestId())
}

func TestCertificateRequesterResponsesWithUnknownIDAreIgnored(t *testing.T) {
	f := newFixture(100 * time.Millisecond)
	defer f.tearDown()

	// Request with different request ID should be ignored.
	go f.respondRequest(t, &central.IssueLocalScannerCertsResponse{RequestId: "UNKNOWN"})

	certs, requestErr := f.requester.RequestCertificates(f.ctx)
	assert.Nil(t, certs)
	assert.Equal(t, context.DeadlineExceeded, requestErr)
}

func TestCertificateRequesterRequestConcurrentRequestDoNotInterfere(t *testing.T) {
	f := newFixture(0)
	defer f.tearDown()
	waitGroup := concurrency.NewWaitGroup(numConcurrentRequests)

	for i := 0; i < numConcurrentRequests; i++ {
		go f.respondRequest(t, nil)
		go func() {
			defer waitGroup.Add(-1)
			_, err := f.requester.RequestCertificates(f.ctx)
			assert.NoError(t, err)
		}()
	}
	ok := concurrency.WaitWithTimeout(&waitGroup, time.Duration(numConcurrentRequests)*testTimeout)
	require.True(t, ok)
}

type certificateRequesterFixture struct {
	sendC                chan *central.MsgFromSensor
	receiveC             chan *central.IssueLocalScannerCertsResponse
	requester            CertificateRequester
	interceptedRequestID *atomic.Value
	ctx                  context.Context
	cancelCtx            context.CancelFunc
}

// newFixture creates a new test fixture that uses `timeout` as context timeout if `timeout` is
// not 0, and `testTimeout` otherwise.
func newFixture(timeout time.Duration) *certificateRequesterFixture {
	sendC := make(chan *central.MsgFromSensor)
	receiveC := make(chan *central.IssueLocalScannerCertsResponse)
	requester := NewCertificateRequester(sendC, receiveC)
	var interceptedRequestID atomic.Value
	if timeout == 0 {
		timeout = testTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	requester.Start()
	return &certificateRequesterFixture{
		sendC:                sendC,
		receiveC:             receiveC,
		requester:            requester,
		ctx:                  ctx,
		cancelCtx:            cancel,
		interceptedRequestID: &interceptedRequestID,
	}
}

func (f *certificateRequesterFixture) tearDown() {
	f.cancelCtx()
	f.requester.Stop()
}

// respondRequest reads a request from `f.sendC` and responds with `responseOverwrite` if not nil, or with
// a response with the same ID as the request otherwise.
// Before sending the response, it stores in `f.interceptedRequestID` the request ID for the requests read from `f.sendC`.
func (f *certificateRequesterFixture) respondRequest(t *testing.T, responseOverwrite *central.IssueLocalScannerCertsResponse) {
	select {
	case <-f.ctx.Done():
	case request := <-f.sendC:
		interceptedRequestID := request.GetIssueLocalScannerCertsRequest().GetRequestId()
		assert.NotEmpty(t, interceptedRequestID)
		var response *central.IssueLocalScannerCertsResponse
		if responseOverwrite != nil {
			response = responseOverwrite
		} else {
			response = &central.IssueLocalScannerCertsResponse{RequestId: interceptedRequestID}
		}
		f.interceptedRequestID.Store(response.GetRequestId())
		select {
		case <-f.ctx.Done():
		case f.receiveC <- response:
		}
	}
}
