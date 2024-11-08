package certificates

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
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

// Response represents the response to a certificate request. It contains a set of certificates or an error.
type Response struct {
	RequestId    string
	ErrorMessage *string
	Certificates *storage.TypedServiceCertificateSet
}

type GenericRequester[ReqT any, ResT ProtobufResponse] struct {
	sendC           chan<- *message.ExpiringMessage
	receiveC        <-chan ResT
	stopC           concurrency.ErrorSignal
	requests        sync.Map
	messageFactory  MessageFactory[ReqT]
	responseFactory ResponseFactory[ResT]
}

type ProtobufResponse interface {
	GetRequestId() string
}

type MessageFactory[ReqT any] interface {
	NewMsgFromSensor(requestID string) *central.MsgFromSensor
}

type ResponseFactory[ResT any] interface {
	ConvertToResponse(response ResT) *Response
}

type LocalScannerResponseFactory struct{}

func (f *LocalScannerResponseFactory) ConvertToResponse(response *central.IssueLocalScannerCertsResponse) *Response {
	return NewResponseFromLocalScannerCerts(response)
}

type SecuredClusterResponseFactory struct{}

func (f *SecuredClusterResponseFactory) ConvertToResponse(response *central.IssueSecuredClusterCertsResponse) *Response {
	return NewResponseFromSecuredClusterCerts(response)
}

type SecuredClusterMessageFactory struct{}

func (f *SecuredClusterMessageFactory) NewMsgFromSensor(requestID string) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_IssueSecuredClusterCertsRequest{
			IssueSecuredClusterCertsRequest: &central.IssueSecuredClusterCertsRequest{
				RequestId: requestID,
			},
		},
	}
}

type LocalScannerMessageFactory struct{}

func (f *LocalScannerMessageFactory) NewMsgFromSensor(requestID string) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_IssueLocalScannerCertsRequest{
			IssueLocalScannerCertsRequest: &central.IssueLocalScannerCertsRequest{
				RequestId: requestID,
			},
		},
	}
}

func NewRequester[ReqT any, ResT ProtobufResponse](
	sendC chan<- *message.ExpiringMessage,
	receiveC <-chan ResT,
	messageFactory MessageFactory[ReqT],
	responseFactory ResponseFactory[ResT],
) *GenericRequester[ReqT, ResT] {
	return &GenericRequester[ReqT, ResT]{
		sendC:           sendC,
		receiveC:        receiveC,
		messageFactory:  messageFactory,
		responseFactory: responseFactory,
	}
}

func (r *GenericRequester[ReqT, ResT]) Start() {
	r.stopC.Reset()
	go r.dispatchResponses()
}

func (r *GenericRequester[ReqT, ResT]) Stop() {
	r.stopC.Signal()
}

func (r *GenericRequester[ReqT, ResT]) dispatchResponses() {
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
			responseC.(chan ResT) <- msg
		}
	}
}

func (r *GenericRequester[ReqT, ResT]) RequestCertificates(ctx context.Context) (*Response, error) {
	requestID := uuid.NewV4().String()
	receiveC := make(chan ResT, 1)
	r.requests.Store(requestID, receiveC)
	defer r.requests.Delete(requestID)

	if err := r.send(ctx, requestID); err != nil {
		return nil, err
	}
	return r.receive(ctx, receiveC)
}

func (r *GenericRequester[ReqT, ResT]) send(ctx context.Context, requestID string) error {
	// Assuming the `message.New` function is generic and can handle different request types.
	msg := r.messageFactory.NewMsgFromSensor(requestID)
	select {
	case <-r.stopC.Done():
		return r.stopC.ErrorWithDefault(ErrCertificateRequesterStopped)
	case <-ctx.Done():
		return ctx.Err()
	case r.sendC <- message.New(msg): // Use a generic `message.New` method for ReqT.
		return nil
	}
}

func (r *GenericRequester[ReqT, ResT]) receive(ctx context.Context, receiveC <-chan ResT) (*Response, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case response := <-receiveC:
		// Convert ResT to `certificates.Response` here, e.g. with a generic conversion function.
		return r.responseFactory.ConvertToResponse(response), nil
	}
}

// NewResponseFromLocalScannerCerts creates a certificates.Response from a
// protobuf central.IssueLocalScannerCertsResponse message
func NewResponseFromLocalScannerCerts(response *central.IssueLocalScannerCertsResponse) *Response {
	if response == nil {
		return nil
	}

	res := &Response{
		RequestId: response.GetRequestId(),
	}

	if response.GetError() != nil {
		errMsg := response.GetError().GetMessage()
		res.ErrorMessage = &errMsg
	} else {
		res.Certificates = response.GetCertificates()
	}

	return res
}

// NewResponseFromSecuredClusterCerts creates a certificates.Response from a
// protobuf central.IssueSecuredClusterCertsResponse message
func NewResponseFromSecuredClusterCerts(response *central.IssueSecuredClusterCertsResponse) *Response {
	if response == nil {
		return nil
	}

	res := &Response{
		RequestId: response.GetRequestId(),
	}

	if response.GetError() != nil {
		errMsg := response.GetError().GetMessage()
		res.ErrorMessage = &errMsg
	} else {
		res.Certificates = response.GetCertificates()
	}

	return res
}
