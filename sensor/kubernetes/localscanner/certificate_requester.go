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

// NewCertificateRequester creates a new certificateRequest manager that communicates through
// the specified channels, and that uses a fresh request ID for reach new request.
// TODO document this handles concurrent requests from several goroutines
func NewCertificateRequester(msgFromSensorC msgFromSensorC, msgToSensorC msgToSensorC) CertificateRequester {
	return &certificateRequesterImpl{
		stopC: concurrency.NewErrorSignal(),
		msgFromSensorC: msgFromSensorC,
		msgToSensorC:   msgToSensorC,
	}
}

type msgFromSensorC chan *central.MsgFromSensor
type msgToSensorC chan *central.IssueLocalScannerCertsResponse
type certificateRequesterImpl struct {
	stopC    concurrency.ErrorSignal
	msgFromSensorC msgFromSensorC
	msgToSensorC   msgToSensorC
	requests sync.Map
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
			}
		}
	}
}

func (m *certificateRequesterImpl) RequestCertificates(ctx context.Context) (*central.IssueLocalScannerCertsResponse, error) {
	certRequester := &certRequestSyncImpl{
		requestID: uuid.NewV4().String(),
		msgFromSensorC: m.msgFromSensorC,
		msgToSensorC: make(msgToSensorC),
	}
	m.requests.Store(certRequester.requestID, certRequester.msgToSensorC)
	response, err := certRequester.requestCertificates(ctx)
	m.requests.Delete(certRequester.requestID)
	return response, err
}