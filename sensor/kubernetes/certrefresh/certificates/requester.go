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
	DispatchResponse(response *Response)
	MsgToCentralC() <-chan *message.ExpiringMessage
}

// NewLocalScannerCertificateRequester creates a new certificate requester for Local Scanner certificates
// (Scanner V2 and Scanner V4). It receives responses through the channel given as parameter,
// and initializes a new request ID for reach request.
// To use it call Start and then make requests with RequestCertificates. Concurrent requests are *not* supported.
func NewLocalScannerCertificateRequester() Requester {
	return newRequester[
		*central.IssueLocalScannerCertsRequest,
		*central.IssueLocalScannerCertsResponse,
	](
		&localScannerMessageFactory{},
		nil,
	)
}

// NewSecuredClusterCertificateRequester creates a new certificate requester for Secured Cluster certificates.
// It receives responses through the channel given as parameter, and initializes a new request ID for reach request.
// To use it call Start and then make requests with RequestCertificates. Concurrent requests are *not* supported.
func NewSecuredClusterCertificateRequester() Requester {
	return newRequester[
		*central.IssueSecuredClusterCertsResponse,
		*central.IssueSecuredClusterCertsResponse,
	](
		&securedClusterMessageFactory{},
		func() *centralsensor.CentralCapability {
			centralCap := centralsensor.CentralCapability(centralsensor.SecuredClusterCertificatesReissue)
			return &centralCap
		}(),
	)
}

func newRequester[ReqT any, RespT protobufResponse](
	messageFactory messageFactory,
	requiredCentralCapability *centralsensor.CentralCapability,
) *genericRequester[ReqT, RespT] {
	return &genericRequester[ReqT, RespT]{
		messageFactory:            messageFactory,
		responseReceived:          concurrency.NewSignal(),
		requiredCentralCapability: requiredCentralCapability,
	}
}

type genericRequester[ReqT any, RespT protobufResponse] struct {
	msgToCentralC             chan *message.ExpiringMessage
	responseFromCentral       *Response
	responseReceived          concurrency.Signal
	stopC                     concurrency.ErrorSignal
	ongoingRequestID          string
	ongoingRequest            atomic.Bool
	messageFactory            messageFactory
	requiredCentralCapability *centralsensor.CentralCapability
	centralChanLock           sync.Mutex
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

// Start starts the certificate requester. Must be called before calling RequestCertificates
func (r *genericRequester[ReqT, RespT]) Start() {
	r.stopC.Reset()

	r.centralChanLock.Lock()
	defer r.centralChanLock.Unlock()
	if r.msgToCentralC == nil {
		r.msgToCentralC = make(chan *message.ExpiringMessage)
	}
}

// Stop stops the certificate requester, and ongoing RequestCertificates calls
func (r *genericRequester[ReqT, RespT]) Stop() {
	r.stopC.Signal()

	r.centralChanLock.Lock()
	defer r.centralChanLock.Unlock()
	if r.msgToCentralC != nil {
		close(r.msgToCentralC)
		r.msgToCentralC = nil
	}
}

// DispatchResponse forwards a response from Central to a RequestCertificates call running in another goroutine
func (r *genericRequester[ReqT, RespT]) DispatchResponse(response *Response) {
	if !r.ongoingRequest.Load() {
		log.Debugf("Received a response for request ID %q, but there is no ongoing RequestCertificates call.", response.RequestId)
		return
	}
	if r.ongoingRequestID != response.RequestId {
		log.Debugf("Request ID %q does not match ongoing request ID %q.",
			response.RequestId, r.ongoingRequestID)
		return
	}

	r.responseFromCentral = response
	r.responseReceived.Signal()
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

	if r.ongoingRequest.Load() {
		return nil, errors.New("concurrent requests are not supported.")
	}
	r.ongoingRequest.Store(true)
	defer func() {
		r.ongoingRequest.Store(false)
	}()

	requestID := uuid.NewV4().String()
	r.ongoingRequestID = requestID
	r.responseReceived.Reset()

	if err := r.send(ctx, requestID); err != nil {
		return nil, err
	}
	return r.receive(ctx)
}

// MsgToCentralC exposes a read channel that contains messages from this component to Central
func (r *genericRequester[ReqT, RespT]) MsgToCentralC() <-chan *message.ExpiringMessage {
	r.centralChanLock.Lock()
	defer r.centralChanLock.Unlock()
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
func (r *genericRequester[ReqT, RespT]) receive(ctx context.Context) (*Response, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-r.stopC.Done():
		return nil, r.stopC.ErrorWithDefault(ErrCertificateRequesterStopped)
	case <-r.responseReceived.Done():
		return r.responseFromCentral, nil
	}
}
