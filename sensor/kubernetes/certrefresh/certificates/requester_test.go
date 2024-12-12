package certificates

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stretchr/testify/assert"
)

var (
	testTimeout = time.Second
)

func TestLocalScannerCertificateRequesterRequestCancellation(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.SecuredClusterCertificatesReissue})
	f := newLocalScannerFixture(0)
	defer f.tearDown()

	f.cancelCtx()
	certs, requestErr := f.requester.RequestCertificates(f.ctx)
	assert.Nil(t, certs)
	assert.Equal(t, context.Canceled, requestErr)
}

func TestSecuredClusterCertificateRequesterRequestCancellation(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.SecuredClusterCertificatesReissue})
	f := newSecuredClusterFixture(0)
	defer f.tearDown()

	f.cancelCtx()
	certs, requestErr := f.requester.RequestCertificates(f.ctx)
	assert.Nil(t, certs)
	assert.Equal(t, context.Canceled, requestErr)
}

func TestLocalScannerCertificateRequesterRequestSuccess(t *testing.T) {
	f := newLocalScannerFixture(0)
	defer f.tearDown()

	go f.respondRequest(t, nil)

	response, err := f.requester.RequestCertificates(f.ctx)
	assert.NoError(t, err)
	assert.Equal(t, f.interceptedRequestID.Load(), response.RequestId)
}

func TestSecuredClusterCertificateRequesterRequestSuccess(t *testing.T) {
	f := newSecuredClusterFixture(0)
	defer f.tearDown()

	go f.respondRequest(t, nil)

	response, err := f.requester.RequestCertificates(f.ctx)
	assert.NoError(t, err)
	assert.Equal(t, f.interceptedRequestID.Load(), response.RequestId)
	oldRequestId := response.RequestId

	// Check that a second call also works
	go f.respondRequest(t, nil)

	response, err = f.requester.RequestCertificates(f.ctx)
	assert.NoError(t, err)
	assert.Equal(t, f.interceptedRequestID.Load(), response.RequestId)
	assert.NotEqual(t, oldRequestId, response.RequestId)
}

func TestLocalScannerCertificateRequesterResponsesWithUnknownIDAreIgnored(t *testing.T) {
	f := newLocalScannerFixture(100 * time.Millisecond)
	defer f.tearDown()

	response := &central.IssueLocalScannerCertsResponse{RequestId: "UNKNOWN"}
	// Request with different request ID should be ignored.
	go f.respondRequest(t, &response)

	certs, requestErr := f.requester.RequestCertificates(f.ctx)
	assert.Nil(t, certs)
	assert.Equal(t, context.DeadlineExceeded, requestErr)
}

func TestSecuredClusterCertificateRequesterResponsesWithUnknownIDAreIgnored(t *testing.T) {
	f := newSecuredClusterFixture(100 * time.Millisecond)
	defer f.tearDown()

	response := &central.IssueSecuredClusterCertsResponse{RequestId: "UNKNOWN"}
	// Request with different request ID should be ignored.
	go f.respondRequest(t, &response)

	certs, requestErr := f.requester.RequestCertificates(f.ctx)
	assert.Nil(t, certs)
	assert.Equal(t, context.DeadlineExceeded, requestErr)
}

func TestSecuredClusterCertificateRequesterNoReplyFromCentral(t *testing.T) {
	f := newSecuredClusterFixture(200 * time.Millisecond)
	defer f.tearDown()

	certs, requestErr := f.requester.RequestCertificates(f.ctx)

	// No response was set using `f.respondRequest`, which simulates not receiving a reply from Central
	assert.Nil(t, certs)
	assert.Equal(t, context.DeadlineExceeded, requestErr)
}

type certificateRequesterFixture[ReqT any, RespT protobufResponse] struct {
	requester            Requester
	interceptedRequestID *atomic.Value
	ctx                  context.Context
	cancelCtx            context.CancelFunc
	getRequestID         requestIDGetter
	newResponseWithID    func(requestID string) RespT
	responseFactory      responseFactory[RespT]
	msgToCentralC        chan *message.ExpiringMessage
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
	var interceptedRequestID atomic.Value
	if timeout == 0 {
		timeout = testTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	fixture := &certificateRequesterFixture[ReqT, RespT]{
		ctx:                  ctx,
		cancelCtx:            cancel,
		interceptedRequestID: &interceptedRequestID,
		getRequestID:         getRequestID,
		newResponseWithID:    newResponseWithID,
		responseFactory:      responseFactory,
		msgToCentralC:        make(chan *message.ExpiringMessage),
	}

	requester := newRequester[ReqT, RespT](messageFactory, nil, fixture.msgToCentralHandler)
	fixture.requester = requester
	return fixture
}

func (f *certificateRequesterFixture[ReqT, RespT]) tearDown() {
	f.cancelCtx()
}

func (f *certificateRequesterFixture[ReqT, RespT]) msgToCentralHandler(ctx context.Context, msg *message.ExpiringMessage) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case f.msgToCentralC <- msg:
		return nil
	}
}

// respondRequest reads a request from `f.MsgToCentralC` and responds with `responseOverwrite` if not nil,
// or with a response with the same ID as the request otherwise.
// Before sending the response, it stores in `f.interceptedRequestID` the ID of the request.
func (f *certificateRequesterFixture[ReqT, RespT]) respondRequest(
	t *testing.T,
	responseOverwrite *RespT) {
	select {
	case <-f.ctx.Done():
	case request := <-f.msgToCentralC:
		interceptedRequestID := f.getRequestID(request)
		assert.NotEmpty(t, interceptedRequestID)
		var response RespT
		if responseOverwrite != nil {
			response = *responseOverwrite
		} else {
			response = f.newResponseWithID(interceptedRequestID)
		}
		f.interceptedRequestID.Store(response.GetRequestId())
		f.requester.DispatchResponse(f.responseFactory.convertToResponse(response))
	}
}

type responseFactory[RespT any] interface {
	convertToResponse(response RespT) *Response
}
type localScannerResponseFactory struct{}

func (f *localScannerResponseFactory) convertToResponse(response *central.IssueLocalScannerCertsResponse) *Response {
	return NewResponseFromLocalScannerCerts(response)
}

type securedClusterResponseFactory struct{}

func (f *securedClusterResponseFactory) convertToResponse(response *central.IssueSecuredClusterCertsResponse) *Response {
	return NewResponseFromSecuredClusterCerts(response)
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
