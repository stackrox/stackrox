package localscanner

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	log                      = logging.LoggerForModule()
	_   CertificateRequester = (*certificateRequesterImpl)(nil)
)

// CertificateRequester request a new set of local scanner certificates to central.
type CertificateRequester interface {
	Start()
	Stop()
	RequestCertificates(ctx context.Context) (*central.IssueLocalScannerCertsResponse, error)
}

// NewCertificateRequester creates a new certificate requester that communicates through
// the specified channels and initializes a new request ID for reach request.
// To use it call Start, and then make requests with RequestCertificates, concurrent requests are supported.
// This assumes that the certificate requester is the only consumer of receiveC.
func NewCertificateRequester(msgFromSensorC chan *central.MsgFromSensor,
	msgToSensorC chan *central.IssueLocalScannerCertsResponse) CertificateRequester {
	return &certificateRequesterImpl{
		stopC:    concurrency.NewErrorSignal(),
		sendC:    msgFromSensorC,
		receiveC: msgToSensorC,
	}
}

type certificateRequesterImpl struct {
	stopC           concurrency.ErrorSignal
	sendC    chan *central.MsgFromSensor
	receiveC chan *central.IssueLocalScannerCertsResponse
	requests sync.Map
}

func (r *certificateRequesterImpl) Start() {
	go r.forwardMessagesToSensor()
}

func (r *certificateRequesterImpl) Stop() {
	r.stopC.Signal()
}

func (r *certificateRequesterImpl) forwardMessagesToSensor() {
	for {
		select {
		case <-r.stopC.Done():
			return
		case msg := <-r.receiveC:
			requestC, ok := r.requests.Load(msg.GetRequestId())
			if ok {
				requestC.(chan *central.IssueLocalScannerCertsResponse) <- msg
			} else {
				log.Debugf("request ID %q does not match any known request ID, skipping request",
					msg.GetRequestId())
			}
		}
	}
}

func (r *certificateRequesterImpl) RequestCertificates(ctx context.Context) (*central.IssueLocalScannerCertsResponse, error) {
	requestID := uuid.NewV4().String()
	receiveC := make(chan *central.IssueLocalScannerCertsResponse)
	r.requests.Store(requestID, receiveC)
	defer r.requests.Delete(requestID)

	if err := r.send(ctx, requestID); err != nil {
		return nil, err
	}
	return receive(ctx, receiveC)
}

func (r *certificateRequesterImpl) send(ctx context.Context, requestID string) error {
	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_IssueLocalScannerCertsRequest{
			IssueLocalScannerCertsRequest: &central.IssueLocalScannerCertsRequest{
				RequestId: requestID,
			},
		},
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case r.sendC <- msg:
		log.Debugf("request to issue local Scanner certificates sent to Central successfully: %v", msg)
		return nil
	}
}

func receive(ctx context.Context, msgToSensorC chan *central.IssueLocalScannerCertsResponse) (*central.IssueLocalScannerCertsResponse, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case response := <-msgToSensorC:
		return response, nil
	}
}
