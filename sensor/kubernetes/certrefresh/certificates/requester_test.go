package certificates

import (
	"context"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/centralcaps"
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

func TestLocalScannerCertificateRequesterRequestFailureIfStopped(t *testing.T) {
	testCases := map[string]struct {
		startRequester bool
	}{
		"requester not started":            {false},
		"requester stopped before request": {true},
	}
	for tcName, tc := range testCases {
		t.Run(tcName, func(t *testing.T) {
			f := newLocalScannerFixture(0)
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

func TestSecuredClusterCertificateRequesterRequestFailureIfStopped(t *testing.T) {
	testCases := map[string]struct {
		startRequester bool
	}{
		"requester not started":            {false},
		"requester stopped before request": {true},
	}
	for tcName, tc := range testCases {
		t.Run(tcName, func(t *testing.T) {
			f := newSecuredClusterFixture(0)
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

func TestLocalScannerCertificateRequesterRequestCancellation(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.SecuredClusterCertificatesReissue})
	f := newLocalScannerFixture(0)
	f.requester.Start()
	defer f.tearDown()

	f.cancelCtx()
	certs, requestErr := f.requester.RequestCertificates(f.ctx)
	assert.Nil(t, certs)
	assert.Equal(t, context.Canceled, requestErr)
}

func TestSecuredClusterCertificateRequesterRequestCancellation(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.SecuredClusterCertificatesReissue})
	f := newSecuredClusterFixture(0)
	f.requester.Start()
	defer f.tearDown()

	f.cancelCtx()
	certs, requestErr := f.requester.RequestCertificates(f.ctx)
	assert.Nil(t, certs)
	assert.Equal(t, context.Canceled, requestErr)
}

func TestLocalScannerCertificateRequesterRequestSuccess(t *testing.T) {
	f := newLocalScannerFixture(0)
	f.requester.Start()
	defer f.tearDown()

	go f.respondRequest(t, 0, nil)

	response, err := f.requester.RequestCertificates(f.ctx)
	assert.NoError(t, err)
	assert.Equal(t, f.interceptedRequestID.Load(), response.RequestId)
}

func TestSecuredClusterCertificateRequesterRequestSuccess(t *testing.T) {
	f := newSecuredClusterFixture(0)
	f.requester.Start()
	defer f.tearDown()

	go f.respondRequest(t, 0, nil)

	response, err := f.requester.RequestCertificates(f.ctx)
	assert.NoError(t, err)
	assert.Equal(t, f.interceptedRequestID.Load(), response.RequestId)
}

func TestLocalScannerCertificateRequesterResponsesWithUnknownIDAreIgnored(t *testing.T) {
	f := newLocalScannerFixture(100 * time.Millisecond)
	f.requester.Start()
	defer f.tearDown()

	var response *central.IssueLocalScannerCertsResponse
	response = &central.IssueLocalScannerCertsResponse{RequestId: "UNKNOWN"}
	// Request with different request ID should be ignored.
	go f.respondRequest(t, 0, &response)

	certs, requestErr := f.requester.RequestCertificates(f.ctx)
	assert.Nil(t, certs)
	assert.Equal(t, context.DeadlineExceeded, requestErr)
}

func TestSecuredClusterCertificateRequesterResponsesWithUnknownIDAreIgnored(t *testing.T) {
	f := newSecuredClusterFixture(100 * time.Millisecond)
	f.requester.Start()
	defer f.tearDown()

	var response *central.IssueSecuredClusterCertsResponse
	response = &central.IssueSecuredClusterCertsResponse{RequestId: "UNKNOWN"}
	// Request with different request ID should be ignored.
	go f.respondRequest(t, 0, &response)

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
			f := newSecuredClusterFixture(0)
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

type certificateRequesterFixture[ReqT any, RespT protobufResponse] struct {
	sendC                chan *message.ExpiringMessage
	receiveC             chan RespT
	requester            Requester
	interceptedRequestID *atomic.Value
	ctx                  context.Context
	cancelCtx            context.CancelFunc
	getRequestID         requestIDGetter
	newResponseWithID    func(requestID string) RespT
}

func newLocalScannerFixture(timeout time.Duration) *certificateRequesterFixture[*central.IssueLocalScannerCertsRequest,
	*central.IssueLocalScannerCertsResponse] {
	return newFixture[*central.IssueLocalScannerCertsRequest, *central.IssueLocalScannerCertsResponse](
		timeout,
		&localScannerMessageFactory{},
		&localScannerResponseFactory{},
		localScannerRequestIDGetter,
		newLocalScannerResponseWithID)
}

func newSecuredClusterFixture(timeout time.Duration) *certificateRequesterFixture[*central.IssueSecuredClusterCertsRequest,
	*central.IssueSecuredClusterCertsResponse] {
	return newFixture[*central.IssueSecuredClusterCertsRequest,
		*central.IssueSecuredClusterCertsResponse](
		timeout,
		&securedClusterMessageFactory{},
		&securedClusterResponseFactory{},
		securedClusterRequestIDGetter,
		newSecuredClusterResponseWithID)
}

// newFixture creates a new test fixture that uses `timeout` as context timeout if `timeout` is
// not 0, and `testTimeout` otherwise.
func newFixture[ReqT any, RespT protobufResponse](
	timeout time.Duration,
	messageFactory messageFactory,
	responseFactory responseFactory[RespT],
	getRequestID requestIDGetter,
	newResponseWithID func(requestID string) RespT,
) *certificateRequesterFixture[ReqT, RespT] {
	sendC := make(chan *message.ExpiringMessage)
	receiveC := make(chan RespT)
	requester := newRequester[ReqT, RespT](sendC, receiveC, messageFactory, responseFactory)
	var interceptedRequestID atomic.Value
	if timeout == 0 {
		timeout = testTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return &certificateRequesterFixture[ReqT, RespT]{
		sendC:                sendC,
		receiveC:             receiveC,
		requester:            requester,
		ctx:                  ctx,
		cancelCtx:            cancel,
		interceptedRequestID: &interceptedRequestID,
		getRequestID:         getRequestID,
		newResponseWithID:    newResponseWithID,
	}
}

func (f *certificateRequesterFixture[ReqT, RespT]) tearDown() {
	f.cancelCtx()
	f.requester.Stop()
}

// respondRequest reads a request from `f.sendC` and responds with `responseOverwrite` if not nil, or with
// a response with the same ID as the request otherwise. If `responseDelay` is greater than 0 then this function
// waits for that time before sending the response.
// Before sending the response, it stores in `f.interceptedRequestID` the request ID for the requests read from `f.sendC`.
func (f *certificateRequesterFixture[ReqT, RespT]) respondRequest(
	t *testing.T,
	responseDelay time.Duration,
	responseOverwrite *RespT) {
	select {
	case <-f.ctx.Done():
	case request := <-f.sendC:
		interceptedRequestID := f.getRequestID(request)
		assert.NotEmpty(t, interceptedRequestID)
		var response RespT
		if responseOverwrite != nil {
			response = *responseOverwrite
		} else {
			response = f.newResponseWithID(interceptedRequestID)
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

type requestIDGetter func(*message.ExpiringMessage) string

func localScannerRequestIDGetter(msg *message.ExpiringMessage) string {
	return msg.GetIssueLocalScannerCertsRequest().GetRequestId()
}

func securedClusterRequestIDGetter(msg *message.ExpiringMessage) string {
	return msg.GetIssueSecuredClusterCertsRequest().GetRequestId()
}

func newLocalScannerResponseWithID(requestID string) *central.IssueLocalScannerCertsResponse {
	return &central.IssueLocalScannerCertsResponse{RequestId: requestID}
}

func newSecuredClusterResponseWithID(requestID string) *central.IssueSecuredClusterCertsResponse {
	return &central.IssueSecuredClusterCertsResponse{RequestId: requestID}
}
