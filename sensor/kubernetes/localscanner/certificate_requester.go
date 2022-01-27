package localscanner

import (
	"context"
	"sync"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
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
// This assumes that the certificate requester is the only consumer of msgToSensorC.
func NewCertificateRequester(msgFromSensorC msgFromSensorC, msgToSensorC msgToSensorC) CertificateRequester {
	return &certificateRequesterImpl{
		stopC:          concurrency.NewErrorSignal(),
		msgFromSensorC: msgFromSensorC,
		msgToSensorC:   msgToSensorC,
	}
}

type msgFromSensorC chan *central.MsgFromSensor
type msgToSensorC chan *central.IssueLocalScannerCertsResponse
type certificateRequesterImpl struct {
	stopC          concurrency.ErrorSignal
	msgFromSensorC msgFromSensorC
	msgToSensorC   msgToSensorC
	requests       sync.Map
}

func (m *certificateRequesterImpl) Start() {
	go m.forwardMessagesToSensor()
}

func (m *certificateRequesterImpl) Stop() {
	m.stopC.Signal()
}

func (m *certificateRequesterImpl) forwardMessagesToSensor() {
	for {
		select {
		case <-m.stopC.Done():
			return
		case msg := <-m.msgToSensorC:
			requestC, ok := m.requests.Load(msg.GetRequestId())
			if ok {
				requestC.(msgToSensorC) <- msg
			} else {
				log.Debugf("request ID %q does not match any known request ID, skipping request",
					msg.GetRequestId()) // FIXME debug
			}
		}
	}
}

func (m *certificateRequesterImpl) RequestCertificates(ctx context.Context) (*central.IssueLocalScannerCertsResponse, error) {
	request := &certRequestSyncImpl{
		requestID:      uuid.NewV4().String(),
		msgFromSensorC: m.msgFromSensorC,
		msgToSensorC:   make(msgToSensorC),
	}
	m.requests.Store(request.requestID, request.msgToSensorC)
	response, err := request.requestCertificates(ctx)
	m.requests.Delete(request.requestID)
	return response, err
}
