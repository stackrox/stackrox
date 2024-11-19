package certificates

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/centralcaps"
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
	MsgToCentralC() <-chan *message.ExpiringMessage
}

// NewLocalScannerCertificateRequester creates a new certificate requester for Local Scanner certificates
// (Scanner V2 and Scanner V4). It receives responses through the channel given as parameter,
// and initializes a new request ID for reach request.
// To use it call Start and then make requests with RequestCertificates. Concurrent requests are supported.
// This assumes that the returned certificate requester is the only consumer of `respFromCentralC`.
func NewLocalScannerCertificateRequester(respFromCentralC <-chan *central.IssueLocalScannerCertsResponse) Requester {
	return newRequester[
		*central.IssueLocalScannerCertsRequest,
		*central.IssueLocalScannerCertsResponse,
	](
		make(chan *message.ExpiringMessage),
		respFromCentralC,
		&localScannerMessageFactory{},
		&localScannerResponseFactory{},
		nil,
	)
}

// NewSecuredClusterCertificateRequester creates a new certificate requester for Secured Cluster certificates.
// It receives responses through the channel given as parameter, and initializes a new request ID for reach request.
// To use it call Start and then make requests with RequestCertificates. Concurrent requests are supported.
// This assumes that the returned certificate requester is the only consumer of `respFromCentralC`.
func NewSecuredClusterCertificateRequester(respFromCentralC <-chan *central.IssueSecuredClusterCertsResponse) Requester {
	return newRequester[
		*central.IssueSecuredClusterCertsResponse,
		*central.IssueSecuredClusterCertsResponse,
	](
		make(chan *message.ExpiringMessage),
		respFromCentralC,
		&securedClusterMessageFactory{},
		&securedClusterResponseFactory{},
		func() *centralsensor.CentralCapability {
			centralCap := centralsensor.CentralCapability(centralsensor.SecuredClusterCertificatesReissue)
			return &centralCap
		}(),
	)
}

func newRequester[ReqT any, RespT protobufResponse](
	msgToCentralC chan *message.ExpiringMessage,
	respFromCentralC <-chan RespT,
	messageFactory messageFactory,
	responseFactory responseFactory[RespT],
	requiredCentralCapability *centralsensor.CentralCapability,
) *genericRequester[ReqT, RespT] {
	return &genericRequester[ReqT, RespT]{
		msgToCentralC:             msgToCentralC,
		respFromCentralC:          respFromCentralC,
		messageFactory:            messageFactory,
		responseFactory:           responseFactory,
		requiredCentralCapability: requiredCentralCapability,
	}
}

type genericRequester[ReqT any, RespT protobufResponse] struct {
	msgToCentralC             chan *message.ExpiringMessage
	respFromCentralC          <-chan RespT
	stopC                     concurrency.ErrorSignal
	requests                  sync.Map
	messageFactory            messageFactory
	responseFactory           responseFactory[RespT]
	requiredCentralCapability *centralsensor.CentralCapability
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

// Start makes the certificate requester listen to `respFromCentralC` and forwards responses to any request that is running
// as a call to RequestCertificates.
func (r *genericRequester[ReqT, RespT]) Start() {
	r.stopC.Reset()
	go r.dispatchResponses()
}

// Stop makes the certificate stop forwarding responses to running requests. Subsequent calls to RequestCertificates
// will fail with ErrCertificateRequesterStopped.
// Currently active calls to RequestCertificates will continue running until cancelled or timed out via the
// provided context.
func (r *genericRequester[ReqT, RespT]) Stop() {
	r.stopC.Signal()
}

// dispatchResponses processes all certificate refresh responses from Central and forwards them
// to their appropriate channel
func (r *genericRequester[ReqT, RespT]) dispatchResponses() {
	for {
		select {
		case <-r.stopC.Done():
			return
		case msg := <-r.respFromCentralC:
			certsResponseC, ok := r.requests.Load(msg.GetRequestId())
			if !ok {
				log.Debugf("Request ID %q does not match any known request ID, dropping response",
					msg.GetRequestId())
				continue
			}
			r.requests.Delete(msg.GetRequestId())
			// Doesn't block even if the corresponding call to RequestCertificates is cancelled and no one
			// ever reads this, because requestC has buffer of 1, and we removed it from `r.requests` above,
			// in case we get more than 1 response for `msg.GetRequestId()`.
			certsResponseC.(chan RespT) <- msg
			close(certsResponseC.(chan RespT))
		}
	}
}

// RequestCertificates makes a new request for a new set of secured cluster certificates from Central.
// This assumes the certificate requester is started, otherwise this returns ErrCertificateRequesterStopped.
func (r *genericRequester[ReqT, RespT]) RequestCertificates(ctx context.Context) (*Response, error) {
	if r.requiredCentralCapability != nil {
		// Central capabilities are only available after this component is created,
		// which is why this check is done here
		if !centralcaps.Has(*r.requiredCentralCapability) {
			return nil, fmt.Errorf("TLS certificate refresh failed: missing Central capability '%s'", *r.requiredCentralCapability)
		}
	}

	// create a new channel for this specific request, and store it in the requests map
	requestID := uuid.NewV4().String()
	certsResponseC := make(chan RespT, 1)
	r.requests.Store(requestID, certsResponseC)

	// Always delete this entry when leaving this scope to account for requests that are never responded, to avoid
	// having entries in `r.requests` that are never removed.
	defer r.requests.Delete(requestID)

	if err := r.send(ctx, requestID); err != nil {
		return nil, err
	}
	return r.receive(ctx, certsResponseC)
}

// MsgToCentralC exposes a read channel that contains messages from this component to Central
func (r *genericRequester[ReqT, RespT]) MsgToCentralC() <-chan *message.ExpiringMessage {
	return r.msgToCentralC
}

// send a cert refresh request to Central
func (r *genericRequester[ReqT, RespT]) send(ctx context.Context, requestID string) error {
	msg := r.messageFactory.newMsgFromSensor(requestID)
	select {
	case <-r.stopC.Done():
		return r.stopC.ErrorWithDefault(ErrCertificateRequesterStopped)
	case <-ctx.Done():
		return ctx.Err()
	case r.msgToCentralC <- message.New(msg):
		return nil
	}
}

// receive handles the response to a specific certificate request
func (r *genericRequester[ReqT, RespT]) receive(ctx context.Context, certsResponseC <-chan RespT) (*Response, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case response := <-certsResponseC:
		return r.responseFactory.convertToResponse(response), nil
	}
}
