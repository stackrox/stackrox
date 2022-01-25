package localscanner

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/sensor/common"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	_  common.SensorComponent = (*localScannerTLSIssuerImpl)(nil)
)

func NewLocalScannerTLSIssuer(certRefreshTimeout time.Duration, certRefreshBackoff wait.Backoff) common.SensorComponent {
	msgFromSensorC := make(msgFromSensorChan)
	msgToSensorC := make(msgToSensorChan)
	requestCertificates := func(ctx context.Context) (*central.IssueLocalScannerCertsResponse, error) {
		certRequester := NewCertificateRequester(msgFromSensorC, msgToSensorC)
		return certRequester.RequestCertificates(ctx)
	}
	return &localScannerTLSIssuerImpl{
		msgFromSensorC: msgFromSensorC,
		msgToSensorC: msgToSensorC,
		certRefresher: newCertRefresher(requestCertificates, certRefreshTimeout, certRefreshBackoff),
	}
}

type msgFromSensorChan chan *central.MsgFromSensor
type msgToSensorChan   chan *central.IssueLocalScannerCertsResponse
type localScannerTLSIssuerImpl struct {
	msgFromSensorC msgFromSensorChan
	msgToSensorC   msgToSensorChan
	certRefresher certRefresher
}

func (i *localScannerTLSIssuerImpl) Start() error {
	log.Info("starting local scanner TLS issuer.")
	i.certRefresher.Start()
	log.Info("local scanner TLS issuer started.")

	return nil
}

func (i *localScannerTLSIssuerImpl) Stop(err error) {
	i.certRefresher.Stop()
	log.Info("local scanner TLS issuer stopped.")
}

func (i *localScannerTLSIssuerImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.LocalScannerCredentialsRefresh}
}

// ResponsesC is called "responses" because for other SensorComponent it is central that
// initiates the interaction. However, here it is sensor which sends a request to central.
func (i *localScannerTLSIssuerImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return i.msgFromSensorC
}

// ProcessMessage is how the central receiver delivers messages from central to SensorComponents.
// This method must not block as it would prevent centralReceiverImpl from sending messages
// to other SensorComponents.
func (i *localScannerTLSIssuerImpl) ProcessMessage(msg *central.MsgToSensor) error {
	switch m := msg.GetMsg().(type) {
	case *central.MsgToSensor_IssueLocalScannerCertsResponse:
		response := m.IssueLocalScannerCertsResponse
		go func() {
			// will block if i.resultC is filled.
			i.msgToSensorC <- response
		}()
		return nil
	default:
		// silently ignore other messages broadcasted by centralReceiverImpl, as centralReceiverImpl logs
		// all returned errors with error level.
		return nil
	}
}