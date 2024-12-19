package certificates

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/message"
)

var (
	log = logging.LoggerForModule()
)

// Requester defines an interface for requesting TLS certificates from Central
type Requester interface {
	RequestCertificates(ctx context.Context) (*Response, error)
	DispatchResponse(response *Response)
}

type MsgToCentralFn func(ctx context.Context, msg *message.ExpiringMessage) error

// NewLocalScannerCertificateRequester creates a new certificate requester for Local Scanner certificates
// (Scanner V2 and Scanner V4).
func NewLocalScannerCertificateRequester(msgToCentralFn MsgToCentralFn) Requester {
	return newRequester[
		*central.IssueLocalScannerCertsRequest,
		*central.IssueLocalScannerCertsResponse,
	](
		&localScannerMessageFactory{},
		nil,
		msgToCentralFn,
	)
}

// NewSecuredClusterCertificateRequester creates a new certificate requester for Secured Cluster certificates.
func NewSecuredClusterCertificateRequester(msgToCentralFn MsgToCentralFn) Requester {
	return newRequester[
		*central.IssueSecuredClusterCertsResponse,
		*central.IssueSecuredClusterCertsResponse,
	](
		&securedClusterMessageFactory{},
		func() *centralsensor.CentralCapability {
			centralCap := centralsensor.CentralCapability(centralsensor.SecuredClusterCertificatesReissue)
			return &centralCap
		}(),
		msgToCentralFn,
	)
}

func newRequester[ReqT any, RespT protobufResponse](
	messageFactory messageFactory,
	requiredCentralCapability *centralsensor.CentralCapability,
	msgToCentralFn MsgToCentralFn,
) *genericRequester[ReqT, RespT] {
	return &genericRequester[ReqT, RespT]{
		messageFactory:            messageFactory,
		responseReceived:          concurrency.NewSignal(),
		requiredCentralCapability: requiredCentralCapability,
		msgToCentralFn:            msgToCentralFn,
	}
}

type genericRequester[ReqT any, RespT protobufResponse] struct {
	msgToCentralFn            MsgToCentralFn
	responseFromCentral       *Response
	responseReceived          concurrency.Signal
	ongoingRequestID          string
	requestOngoing            atomic.Bool
	messageFactory            messageFactory
	requiredCentralCapability *centralsensor.CentralCapability
}

type protobufResponse interface {
	GetRequestId() string
}

type messageFactory interface {
	newMsgFromSensor(requestID string) *central.MsgFromSensor
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

// DispatchResponse forwards a response from Central to a RequestCertificates call running in another goroutine
func (r *genericRequester[ReqT, RespT]) DispatchResponse(response *Response) {
	if !r.requestOngoing.Load() {
		log.Warnf("Received a response for request ID %q, but there is no ongoing RequestCertificates call.", response.RequestId)
		return
	}
	if r.ongoingRequestID != response.RequestId {
		log.Warnf("Request ID %q does not match ongoing request ID %q.",
			response.RequestId, r.ongoingRequestID)
		return
	}

	r.responseFromCentral = response
	r.responseReceived.Signal()
}

// RequestCertificates makes a new request for a new set of secured cluster certificates from Central.
// Concurrent requests are *not* supported.
func (r *genericRequester[ReqT, RespT]) RequestCertificates(ctx context.Context) (*Response, error) {
	if r.requiredCentralCapability != nil {
		// Central capabilities are only available after this component is created,
		// which is why this check is done here
		if !centralcaps.Has(*r.requiredCentralCapability) {
			return nil, fmt.Errorf("TLS certificate refresh failed: missing Central capability '%s'", *r.requiredCentralCapability)
		}
	}

	if !r.requestOngoing.CompareAndSwap(false, true) {
		return nil, errors.New("concurrent requests are not supported.")
	}

	defer func() {
		r.requestOngoing.Store(false)
		r.responseReceived.Reset()
	}()

	requestID := uuid.NewV4().String()
	r.ongoingRequestID = requestID

	if err := r.send(ctx, requestID); err != nil {
		return nil, err
	}
	return r.receive(ctx)
}

// send a cert refresh request to Central
func (r *genericRequester[ReqT, RespT]) send(ctx context.Context, requestID string) error {
	msg := r.messageFactory.newMsgFromSensor(requestID)
	return r.msgToCentralFn(ctx, message.New(msg))
}

// receive handles the response to a specific certificate request
func (r *genericRequester[ReqT, RespT]) receive(ctx context.Context) (*Response, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-r.responseReceived.Done():
		return r.responseFromCentral, nil
	}
}
