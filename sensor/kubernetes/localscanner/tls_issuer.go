package localscanner

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/kubernetes/certificates"
	"k8s.io/apimachinery/pkg/util/wait"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	log = logging.LoggerForModule()
	_ common.SensorComponent = (*localScannerTLSIssuerImpl)(nil)
	_ certificates.CertificateSource = (*localScannerTLSIssuerImpl)(nil)
)

// FIXME separate files for different structs
type localScannerTLSIssuerImpl struct {
	conf config
	certRefresher certificates.CertRefresher
	certificateSourceImpl
	sensorComponentImpl
}

type config struct {
	sensorNamespace  string
	secretsClient corev1.SecretInterface
}

type certificateSourceImpl struct {
	requestID string
	resultC              chan *retry.Result
	// protects both requestID and resultC
	certSourceStateMutex sync.Mutex
	sensorComponentImpl
}

type sensorComponentImpl struct {
	requestsC chan *central.MsgFromSensor
}

/*
TODO create function
    resultC = make(chan *retry.Result)
*/

func (i *localScannerTLSIssuerImpl) Start() error {
	log.Info("starting local scanner TLS issuer.")

	var certRequestBackoff wait.Backoff // FIXME
	i.certRefresher = certificates.NewCertRefresher("FIXME desc", i, certRequestBackoff)
	i.certRefresher.Start(context.Background())

	log.Info("local scanner TLS issuer started.")

	return nil
}

func (i *localScannerTLSIssuerImpl) Stop(err error) {
	if i.certRefresher != nil {
		i.certRefresher.Stop()
	}
	i.Close()
	log.Info("local scanner TLS issuer stopped.")
}

func (i *sensorComponentImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{} // FIXME
}

// ResponsesC is called "responses" because for other SensorComponent it is central that
// initiates the interaction. However, here it is sensor which sends a request to central.
func (i *sensorComponentImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return i.requestsC
}

// ProcessMessage cannot block as it would prevent centralReceiverImpl from sending messages
// to other SensorComponent.
func (i *localScannerTLSIssuerImpl) ProcessMessage(msg *central.MsgToSensor) error {
	switch m := msg.GetMsg().(type) {
	case *central.MsgToSensor_IssueLocalScannerCertsResponse:
		response := m.IssueLocalScannerCertsResponse
		go func() {
			i.processIssueLocalScannerCertsResponse(response)
		}()
		return nil
	default:
		// silently ignore other messages broadcasted by centralReceiverImpl, as centralReceiverImpl logs
		// all returned errors with error level.
		return nil
	}
}

func (i *certificateSourceImpl) processIssueLocalScannerCertsResponse(response *central.IssueLocalScannerCertsResponse) {
	i.certSourceStateMutex.Lock()
	defer i.certSourceStateMutex.Unlock()

	if response.GetRequestId() != i.requestID {
		log.Debugf("ignoring response with unknown request id %s", response.GetRequestId())
		return
	}
	i.requestID = ""
	var result *retry.Result
	if response.GetError() != nil {
		result = &retry.Result{Err: errors.Errorf("server side error: %s", response.GetError().GetMessage())}
	} else {
		// retry.Result is untyped, so at least type here.
		var certificates *storage.TypedServiceCertificateSet
		certificates = response.GetCertificates()
		result = &retry.Result{ V: certificates }
	}
	resultC := i.resultC
	go func() {
		//  can block if i.resultC is filled.
		resultC <- result
	}()
}

func (i *certificateSourceImpl) AskForResult(ctx context.Context) <-chan *retry.Result {
	resultC := i.resetCertSource()
	go func() {
		i.Retry()
	}()

	return resultC
}

// Retry blocks until the message is sent or we get a timeout.
func (i *certificateSourceImpl) Retry() {
	i.certSourceStateMutex.Lock()
	defer i.certSourceStateMutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second) // FIXME timeout and context, or retry with backoff in a goroutine
	defer cancel()

	requestID := uuid.NewV4().String()
	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_IssueLocalScannerCertsRequest{
			IssueLocalScannerCertsRequest: &central.IssueLocalScannerCertsRequest{
				RequestId: requestID,
			},
		},
	}
	select {
	case i.requestsC <- msg:
		i.requestID = requestID
		log.Debugf("request to issue local Scanner certificates sent to Central succesfully: %v", msg)
	case <-ctx.Done():
		i.requestID = ""
		resultC := i.resultC
		go func() {
			//  can block if i.resultC is filled.
			resultC <- &retry.Result{ Err: errors.Wrap(ctx.Err(), "sending the request to central") }
		}()
	}
}

func (i *certificateSourceImpl) Close() {
	i.certSourceStateMutex.Lock()
	defer i.certSourceStateMutex.Unlock()

	oldResultC := i.resultC
	i.doResetCertSource()
	go func() {
		if oldResultC != nil {
			// drain channel in case the reader gave up, to avoid
			// zombie goroutines.
			for {
				select {
				case <-oldResultC:
				default:
					break
				}
			}
		}
	}()
}

func (i *certificateSourceImpl) HandleCertificates(certificates *storage.TypedServiceCertificateSet) (timeToRefresh time.Duration, err error) {
	// TODO get secrets => secretRepository type
	if certificates != nil {
		// TODO update and store secrets => secretRepository type
	}
	// TODO get duration from secrets => secretExpirationStrategy type
	return time.Minute, nil // FIXME
}

func (i *certificateSourceImpl) resetCertSource() chan *retry.Result {
	i.certSourceStateMutex.Lock()
	defer i.certSourceStateMutex.Unlock()

	i.doResetCertSource()
	return i.resultC
}
func (i *certificateSourceImpl) doResetCertSource() {
	i.requestID = ""
	i.resultC = make(chan *retry.Result)
}
