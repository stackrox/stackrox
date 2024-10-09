package localscanner

import (
	"context"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	numConcurrentRequests = 10
)

var (
	testTimeout = time.Second
)

func TestCertificateRequesterRequestFailureIfStopped(t *testing.T) {
	testCases := map[string]struct {
		startRequester bool
	}{
		"requester not started":            {false},
		"requester stopped before request": {true},
	}
	for tcName, tc := range testCases {
		t.Run(tcName, func(t *testing.T) {
			f := newFixture(0)
			defer f.tearDown()
			if tc.startRequester {
				f.requester.Start()
				f.requester.Stop()
			}

			certs, requestErr := f.requester.RequestCertificates(f.ctx)
			assert.Nil(t, certs)
			assert.Equal(t, ErrCertificateRequesterStopped, requestErr)
		})
	}
}

func TestCertificateRequesterRequestCancellation(t *testing.T) {
	f := newFixture(0)
	f.requester.Start()
	defer f.tearDown()

	f.cancelCtx()
	certs, requestErr := f.requester.RequestCertificates(f.ctx)
	assert.Nil(t, certs)
	assert.Equal(t, context.Canceled, requestErr)
}

func TestCertificateRequesterRequestSuccess(t *testing.T) {
	f := newFixture(0)
	f.requester.Start()
	defer f.tearDown()

	go f.respondRequest(t, 0, nil)

	response, err := f.requester.RequestCertificates(f.ctx)
	assert.NoError(t, err)
	assert.Equal(t, f.interceptedRequestID.Load(), response.GetRequestId())
}

func TestCertificateRequesterResponsesWithUnknownIDAreIgnored(t *testing.T) {
	f := newFixture(100 * time.Millisecond)
	f.requester.Start()
	defer f.tearDown()

	// Request with different request ID should be ignored.
	go f.respondRequest(t, 0, &central.IssueLocalScannerCertsResponse{RequestId: "UNKNOWN"})

	certs, requestErr := f.requester.RequestCertificates(f.ctx)
	assert.Nil(t, certs)
	assert.Equal(t, context.DeadlineExceeded, requestErr)
}

func TestCertificateRequesterRequestConcurrentRequestDoNotInterfere(t *testing.T) {
	testCases := map[string]struct {
		responseDelayFunc func(requestIndex int) (responseDelay time.Duration)
	}{
		"decreasing response delay": {func(requestIndex int) (responseDelay time.Duration) {
			// responses are responded increasingly faster, so always out of order.
			return time.Duration(numConcurrentRequests-(requestIndex+1)) * 10 * time.Millisecond
		}},
		"random response delay": {func(requestIndex int) (responseDelay time.Duration) {
			// randomly out of order responses.
			return time.Duration(rand.Intn(100)) * time.Millisecond
		}},
	}
	for tcName, tc := range testCases {
		t.Run(tcName, func(t *testing.T) {
			f := newFixture(0)
			f.requester.Start()
			defer f.tearDown()
			waitGroup := concurrency.NewWaitGroup(numConcurrentRequests)

			for i := 0; i < numConcurrentRequests; i++ {
				i := i
				responseDelay := tc.responseDelayFunc(i)
				go f.respondRequest(t, responseDelay, nil)
				go func() {
					defer waitGroup.Add(-1)
					_, err := f.requester.RequestCertificates(f.ctx)
					assert.NoError(t, err)
				}()
			}
			ok := concurrency.WaitWithTimeout(&waitGroup, time.Duration(numConcurrentRequests)*testTimeout)
			require.True(t, ok)
		})
	}
}

type certificateRequesterFixture struct {
	sendC                chan *message.ExpiringMessage
	receiveC             chan *central.IssueLocalScannerCertsResponse
	requester            CertificateRequester
	interceptedRequestID *atomic.Value
	ctx                  context.Context
	cancelCtx            context.CancelFunc
}

// newFixture creates a new test fixture that uses `timeout` as context timeout if `timeout` is
// not 0, and `testTimeout` otherwise.
func newFixture(timeout time.Duration) *certificateRequesterFixture {
	sendC := make(chan *message.ExpiringMessage)
	receiveC := make(chan *central.IssueLocalScannerCertsResponse)
	requester := NewCertificateRequester(sendC, receiveC)
	var interceptedRequestID atomic.Value
	if timeout == 0 {
		timeout = testTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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
// a response with the same ID as the request otherwise. If `responseDelay` is greater than 0 then this function
// waits for that time before sending the response.
// Before sending the response, it stores in `f.interceptedRequestID` the request ID for the requests read from `f.sendC`.
func (f *certificateRequesterFixture) respondRequest(t *testing.T, responseDelay time.Duration, responseOverwrite *central.IssueLocalScannerCertsResponse) {
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
		if responseDelay > 0 {
			select {
			case <-f.ctx.Done():
				return
			case <-time.After(responseDelay):
			}
		}
		select {
		case <-f.ctx.Done():
		case f.receiveC <- response:
		}
	}
}
