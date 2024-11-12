package certificates

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/message"
)

var (
	log                            = logging.LoggerForModule()
	ErrCertificateRequesterStopped = errors.New("certificate requester is stopped")
)

// Requester defines an interface for requesting TLS certificates from Central
type Requester interface {
	Start()
	Stop()
	RequestCertificates(ctx context.Context) (*Response, error)
}

// NewLocalScannerCertificateRequester creates a new local scanner certificate requester that communicates through
// the specified channels and initializes a new request ID for reach request.
// To use it call Start, and then make requests with RequestCertificates, concurrent requests are supported.
// This assumes that the returned certificate requester is the only consumer of `receiveC`.
func NewLocalScannerCertificateRequester(sendC chan<- *message.ExpiringMessage,
	receiveC <-chan *central.IssueLocalScannerCertsResponse) Requester {
	return newRequester[
		*central.IssueLocalScannerCertsRequest,
		*central.IssueLocalScannerCertsResponse,
	](
		sendC,
		receiveC,
		&localScannerMessageFactory{},
		&localScannerResponseFactory{},
	)
}

// NewSecuredClusterCertificateRequester creates a new certificate requester that communicates through
// the specified channels and initializes a new request ID for reach request.
// To use it call Start, and then make requests with RequestCertificates, concurrent requests are supported.
// This assumes that the returned certificate requester is the only consumer of `receiveC`.
func NewSecuredClusterCertificateRequester(sendC chan<- *message.ExpiringMessage,
	receiveC <-chan *central.IssueSecuredClusterCertsResponse) Requester {
	return newRequester[
		*central.IssueSecuredClusterCertsResponse,
		*central.IssueSecuredClusterCertsResponse,
	](
		sendC,
		receiveC,
		&securedClusterMessageFactory{},
		&securedClusterResponseFactory{},
	)
}

func newRequester[ReqT any, RespT protobufResponse](
	sendC chan<- *message.ExpiringMessage,
	receiveC <-chan RespT,
	messageFactory messageFactory,
	responseFactory responseFactory[RespT],
) *genericRequester[ReqT, RespT] {
	return &genericRequester[ReqT, RespT]{
		sendC:           sendC,
		receiveC:        receiveC,
		messageFactory:  messageFactory,
		responseFactory: responseFactory,
	}
}

type genericRequester[ReqT any, RespT protobufResponse] struct {
	sendC           chan<- *message.ExpiringMessage
	receiveC        <-chan RespT
	stopC           concurrency.ErrorSignal
	requests        sync.Map
	messageFactory  messageFactory
	responseFactory responseFactory[RespT]
}

type protobufResponse interface {
	GetRequestId() string
}

type messageFactory interface {
	newMsgFromSensor(requestID string) *central.MsgFromSensor
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

type securedClusterMessageFactory struct{}

func (f *securedClusterMessageFactory) newMsgFromSensor(requestID string) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_IssueSecuredClusterCertsRequest{
			IssueSecuredClusterCertsRequest: &central.IssueSecuredClusterCertsRequest{
				RequestId: requestID,
			},
		},
	}
}

type localScannerMessageFactory struct{}

func (f *localScannerMessageFactory) newMsgFromSensor(requestID string) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_IssueLocalScannerCertsRequest{
			IssueLocalScannerCertsRequest: &central.IssueLocalScannerCertsRequest{
				RequestId: requestID,
			},
		},
	}
}

func (r *genericRequester[ReqT, RespT]) Start() {
	r.stopC.Reset()
	go r.dispatchResponses()
}

func (r *genericRequester[ReqT, RespT]) Stop() {
	r.stopC.Signal()
}

func (r *genericRequester[ReqT, RespT]) dispatchResponses() {
	for {
		select {
		case <-r.stopC.Done():
			return
		case msg := <-r.receiveC:
			responseC, ok := r.requests.Load(msg.GetRequestId())
			if !ok {
				log.Debugf("request ID %q does not match any known request ID, dropping response",
					msg.GetRequestId())
				continue
			}
			r.requests.Delete(msg.GetRequestId())
			// Doesn't block even if the corresponding call to RequestCertificates is cancelled and no one
			// ever reads this, because requestC has buffer of 1, and we removed it from `r.requests` above,
			// in case we get more than 1 response for `msg.GetRequestId()`.
			responseC.(chan RespT) <- msg
		}
	}
}

func (r *genericRequester[ReqT, RespT]) RequestCertificates(ctx context.Context) (*Response, error) {
	requestID := uuid.NewV4().String()
	receiveC := make(chan RespT, 1)
	r.requests.Store(requestID, receiveC)
	defer r.requests.Delete(requestID)

	if err := r.send(ctx, requestID); err != nil {
		return nil, err
	}
	return r.receive(ctx, receiveC)
}

func (r *genericRequester[ReqT, RespT]) send(ctx context.Context, requestID string) error {
	// Assuming the `message.New` function is generic and can handle different request types.
	msg := r.messageFactory.newMsgFromSensor(requestID)
	select {
	case <-r.stopC.Done():
		return r.stopC.ErrorWithDefault(ErrCertificateRequesterStopped)
	case <-ctx.Done():
		return ctx.Err()
	case r.sendC <- message.New(msg): // Use a generic `message.New` method for ReqT.
		return nil
	}
}

func (r *genericRequester[ReqT, RespT]) receive(ctx context.Context, receiveC <-chan RespT) (*Response, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case response := <-receiveC:
		// Convert RespT to `certificates.Response` here, e.g. with a generic conversion function.
		return r.responseFactory.convertToResponse(response), nil
	}
}
