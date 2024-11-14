package securedcluster

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/certificates"
)

var (
	log = logging.LoggerForModule()

	// ErrCertificateRequesterStopped is returned by RequestCertificates when the certificate
	// requester is not initialized.
	ErrCertificateRequesterStopped                        = errors.New("certificate requester stopped")
	_                              certificates.Requester = (*certificateRequesterImpl)(nil)
)

// NewCertificateRequester creates a new Secured Cluster certificates requester that communicates through
// the specified channels and initializes a new request ID for reach request.
// To use it call Start, and then make requests with RequestCertificates, concurrent requests are supported.
// This assumes that the returned certificate requester is the only consumer of `receiveC`.
func NewCertificateRequester(sendC chan<- *message.ExpiringMessage,
	receiveC <-chan *central.IssueSecuredClusterCertsResponse) certificates.Requester {
	return &certificateRequesterImpl{
		sendC:    sendC,
		receiveC: receiveC,
	}
}

// certificateRequesterImpl implements the certificates.Requester interface for Secured Cluster certificates.
// Note that there is a nearly identical certificateRequesterImpl in the localscanner package (using a different
// gRPC API), and any changes or fixes in this file should be carried over to that file as well.
type certificateRequesterImpl struct {
	sendC    chan<- *message.ExpiringMessage
	receiveC <-chan *central.IssueSecuredClusterCertsResponse
	stopC    concurrency.ErrorSignal
	requests sync.Map
}

// Start makes the certificate requester listen to `receiveC` and forwards responses to any request that is running
// as a call to RequestCertificates.
func (r *certificateRequesterImpl) Start() {
	r.stopC.Reset()
	go r.dispatchResponses()
}

// Stop makes the certificate stop forwarding responses to running requests. Subsequent calls to RequestCertificates
// will fail with ErrCertificateRequesterStopped.
// Currently active calls to RequestCertificates will continue running until cancelled or timed out via the
// provided context.
func (r *certificateRequesterImpl) Stop() {
	r.stopC.Signal()
}

func (r *certificateRequesterImpl) dispatchResponses() {
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
			responseC.(chan *central.IssueSecuredClusterCertsResponse) <- msg
		}
	}
}

// RequestCertificates makes a new request for a new set of secured cluster certificates from Central.
// This assumes the certificate requester is started, otherwise this returns ErrCertificateRequesterStopped.
func (r *certificateRequesterImpl) RequestCertificates(ctx context.Context) (*certificates.Response, error) {
	// Central capabilities are only available after this component is created,
	// which is why this check is done here
	if !centralcaps.Has(centralsensor.SecuredClusterCertificatesReissue) {
		return nil, errors.New("TLS certificate refresh failed: missing Central capability 'SecuredClusterCertificatesReissue'")
	}

	requestID := uuid.NewV4().String()
	receiveC := make(chan *central.IssueSecuredClusterCertsResponse, 1)
	r.requests.Store(requestID, receiveC)
	// Always delete this entry when leaving this scope to account for requests that are never responded, to avoid
	// having entries in `r.requests` that are never removed.
	defer r.requests.Delete(requestID)

	if err := r.send(ctx, requestID); err != nil {
		return nil, err
	}
	return receive(ctx, receiveC)
}

func (r *certificateRequesterImpl) send(ctx context.Context, requestID string) error {
	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_IssueSecuredClusterCertsRequest{
			IssueSecuredClusterCertsRequest: &central.IssueSecuredClusterCertsRequest{
				RequestId: requestID,
			},
		},
	}
	select {
	case <-r.stopC.Done():
		return r.stopC.ErrorWithDefault(ErrCertificateRequesterStopped)
	case <-ctx.Done():
		return ctx.Err()
	case r.sendC <- message.New(msg):
		return nil
	}
}

func receive(ctx context.Context, receiveC <-chan *central.IssueSecuredClusterCertsResponse) (*certificates.Response, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case response := <-receiveC:
		return certificates.NewResponseFromSecuredClusterCerts(response), nil
	}
}
