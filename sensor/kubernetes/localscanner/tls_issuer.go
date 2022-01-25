package localscanner

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/kubernetes/certificates"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	log                                = logging.LoggerForModule()
	_   common.SensorComponent         = (*localScannerTLSIssuerImpl)(nil)
	_   certificates.CertificateIssuer = (*localScannerTLSIssuerImpl)(nil)
)

// FIXME separate files for different structs
type localScannerTLSIssuerImpl struct {
	conf          config
	certRefresher certificates.CertRefresher
	certIssuerImpl
	sensorComponentImpl
}

type config struct {
	// TODO sensorNamespace string
	certRefresherBackoff wait.Backoff
}

type sensorComponentImpl struct {
	requestsC  chan *central.MsgFromSensor
	responsesC chan *central.IssueLocalScannerCertsResponse
}

type certIssuerImpl struct {
	sensorComponentImpl
	certSecretsRepoImpl
}

type certSyncRequesterImpl struct {
	requestID  string
	requestsC  chan *central.MsgFromSensor
	responsesC chan *central.IssueLocalScannerCertsResponse
}

type certSecretsRepoImpl struct {
	// TODO secretsClient   corev1.SecretInterface
}

func (i *localScannerTLSIssuerImpl) Start() error {
	log.Info("starting local scanner TLS issuer.")

	i.certRefresher = certificates.NewCertRefresher("FIXME desc", i, i.conf.certRefresherBackoff)
	if err := i.certRefresher.Start(); err != nil {
		return err
	}

	log.Info("local scanner TLS issuer started.")

	return nil
}

func (i *localScannerTLSIssuerImpl) Stop(err error) {
	if i.certRefresher != nil {
		i.certRefresher.Stop()
	}
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
// to other SensorComponent. This is running in the goroutine launched in centralReceiverImpl.Start.
func (i *localScannerTLSIssuerImpl) ProcessMessage(msg *central.MsgToSensor) error {
	switch m := msg.GetMsg().(type) {
	case *central.MsgToSensor_IssueLocalScannerCertsResponse:
		response := m.IssueLocalScannerCertsResponse
		go func() {
			// will block if i.resultC is filled.
			i.responsesC <- response
		}()
		return nil
	default:
		// silently ignore other messages broadcasted by centralReceiverImpl, as centralReceiverImpl logs
		// all returned errors with error level.
		return nil
	}
}

// RefreshCertificates TODO doc
// This is running in the goroutine for a refresh timer in i.certRefresher.
func (i *certIssuerImpl) RefreshCertificates(ctx context.Context) (timeToRefresh time.Duration, err error) {
	secrets, fetchErr := i.fetchSecrets(ctx)
	if fetchErr != nil {
		return 0, fetchErr
	}
	timeToRefresh = time.Until(i.getCertRenewalTime(secrets))
	if timeToRefresh > 0 {
		return timeToRefresh, nil
	}

	certRequester := &certSyncRequesterImpl{
		requestID:  uuid.NewV4().String(),
		requestsC:  i.requestsC,
		responsesC: i.responsesC,
	}
	response, requestErr := certRequester.requestCertificates(ctx)
	if requestErr != nil {
		return 0, requestErr
	}
	if response.GetError() != nil {
		return 0, errors.Errorf("server side error: %s", response.GetError().GetMessage())
	}

	certificates := response.GetCertificates()
	if refreshErr := i.refreshSecrets(ctx, certificates, secrets); refreshErr != nil {
		return 0, refreshErr
	}
	timeToRefresh = time.Until(i.getCertRenewalTime(secrets))
	return timeToRefresh, nil
}

func (i *certIssuerImpl) getCertRenewalTime(secrets map[storage.ServiceType]*v1.Secret) time.Time {
	return time.Now() // TODO
}

func (i *certIssuerImpl) refreshSecrets(ctx context.Context, certificates *storage.TypedServiceCertificateSet,
	secrets map[storage.ServiceType]*v1.Secret) error {
	// TODO
	return i.updateSecrets(secrets)
}

func (i *certSyncRequesterImpl) requestCertificates(ctx context.Context) (*central.IssueLocalScannerCertsResponse, error) {
	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_IssueLocalScannerCertsRequest{
			IssueLocalScannerCertsRequest: &central.IssueLocalScannerCertsRequest{
				RequestId: i.requestID,
			},
		},
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case i.requestsC <- msg:
		log.Debugf("request to issue local Scanner certificates sent to Central succesfully: %v", msg)
	}

	var response *central.IssueLocalScannerCertsResponse
	for response == nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case newResponse := <-i.responsesC:
			if newResponse.GetRequestId() != i.requestID {
				log.Debugf("ignoring response with unknown request id %s", response.GetRequestId())
			} else {
				response = newResponse
			}
		}
	}

	return response, nil
}

func (i *certSecretsRepoImpl) fetchSecrets(ctx context.Context) (map[storage.ServiceType]*v1.Secret, error) {
	secretsMap := make(map[storage.ServiceType]*v1.Secret, 3)
	return secretsMap, nil // TODO
}

func (i *certSecretsRepoImpl) updateSecrets(secrets map[storage.ServiceType]*v1.Secret) error {
	return nil // TODO
}
