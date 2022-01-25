package localscanner

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	log                        = logging.LoggerForModule()
	_   common.SensorComponent = (*localScannerTLSIssuerImpl)(nil)
	//	_   certificates.CertificateIssuer = (*localScannerTLSIssuerImpl)(nil)
	//	_   certificates.CertificateIssuer = (*certIssuerImpl)(nil)
	_ certSecretsRepo = (*certSecretsRepoImpl)(nil)
)

// FIXME separate files for different structs
type localScannerTLSIssuerImpl struct {
	conf              config
	certRefreshTicker *concurrency.RetryTicker
	certIssuerImpl
	sensorComponentImpl
}

type config struct {
	// TODO sensorNamespace string
	certRefresherBackoff wait.Backoff
}

type sensorComponentImpl struct {
	msgFromSensorC chan *central.MsgFromSensor
	msgToSensorC   chan *central.IssueLocalScannerCertsResponse
}

type certIssuerImpl struct {
	sensorComponentImpl
	certSecretsRepoImpl
}

type certSyncRequesterImpl struct {
	requestID      string
	msgFromSensorC chan *central.MsgFromSensor
	msgToSensorC   chan *central.IssueLocalScannerCertsResponse
}

type certSecretsRepo interface {
	getSecrets(ctx context.Context) (map[storage.ServiceType]*v1.Secret, error)
	putSecrets(ctx context.Context, secrets map[storage.ServiceType]*v1.Secret) error
}

type certSecretsRepoImpl struct {
	// TODO secretsClient   corev1.SecretInterface
}

func (i *localScannerTLSIssuerImpl) Start() error {
	log.Info("starting local scanner TLS issuer.")

	i.certRefreshTicker = concurrency.NewRetryTicker(i.RefreshCertificates,
		time.Second,
		i.conf.certRefresherBackoff)
	certsDescription := "local scanner credentials"
	i.certRefreshTicker.OnTickSuccess = func(nextTimeToTick time.Duration) {
		log.Infof("successfully refreshed %v", certsDescription)
		log.Infof("%v scheduled to be refreshed in %s", certsDescription, nextTimeToTick)
	}
	i.certRefreshTicker.OnTickError = func(refreshErr error) {
		log.Errorf("refreshing %s: %s", certsDescription, refreshErr)
	}
	i.certRefreshTicker.Start()

	log.Info("local scanner TLS issuer started.")

	return nil
}

func (i *localScannerTLSIssuerImpl) Stop(err error) {
	if i.certRefreshTicker != nil {
		i.certRefreshTicker.Stop()
	}
	log.Info("local scanner TLS issuer stopped.")
}

func (i *sensorComponentImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{} // FIXME
}

// ResponsesC is called "responses" because for other SensorComponent it is central that
// initiates the interaction. However, here it is sensor which sends a request to central.
func (i *sensorComponentImpl) ResponsesC() <-chan *central.MsgFromSensor {
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

// RefreshCertificates TODO doc
// This is running in the goroutine for a refresh timer in i.certRefresher.
func (i *certIssuerImpl) RefreshCertificates(ctx context.Context) (timeToRefresh time.Duration, err error) {
	secrets, fetchErr := i.getSecrets(ctx)
	if fetchErr != nil {
		return 0, fetchErr
	}
	timeToRefresh = time.Until(i.getCertRenewalTime(secrets))
	if timeToRefresh > 0 {
		return timeToRefresh, nil
	}

	certRequester := &certSyncRequesterImpl{
		requestID:      uuid.NewV4().String(),
		msgFromSensorC: i.msgFromSensorC,
		msgToSensorC:   i.msgToSensorC,
	}
	response, requestErr := certRequester.requestCertificates(ctx)
	if requestErr != nil {
		return 0, requestErr
	}
	if response.GetError() != nil {
		return 0, errors.Errorf("central refused to issue certificates: %s", response.GetError().GetMessage())
	}

	certificates := response.GetCertificates()
	if refreshErr := i.updateSecrets(ctx, certificates, secrets); refreshErr != nil {
		return 0, refreshErr
	}
	timeToRefresh = time.Until(i.getCertRenewalTime(secrets))
	return timeToRefresh, nil
}

func (i *certIssuerImpl) getCertRenewalTime(secrets map[storage.ServiceType]*v1.Secret) time.Time {
	return time.Now() // TODO
}

// updateSecrets stores the certificates in the data of each corresponding secret, and then persists
// the secrets in k8s.
func (i *certIssuerImpl) updateSecrets(ctx context.Context, certificates *storage.TypedServiceCertificateSet,
	secrets map[storage.ServiceType]*v1.Secret) error {
	// TODO
	return i.putSecrets(ctx, secrets)
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
	case i.msgFromSensorC <- msg:
		log.Debugf("request to issue local Scanner certificates sent to Central succesfully: %v", msg)
	}

	var response *central.IssueLocalScannerCertsResponse
	for response == nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case newResponse := <-i.msgToSensorC:
			if newResponse.GetRequestId() != i.requestID {
				log.Debugf("ignoring response with unknown request id %s", response.GetRequestId())
			} else {
				response = newResponse
			}
		}
	}

	return response, nil
}

func (i *certSecretsRepoImpl) getSecrets(ctx context.Context) (map[storage.ServiceType]*v1.Secret, error) {
	secretsMap := make(map[storage.ServiceType]*v1.Secret, 3)
	return secretsMap, nil // TODO
}

func (i *certSecretsRepoImpl) putSecrets(ctx context.Context, secrets map[storage.ServiceType]*v1.Secret) error {
	return nil // TODO
}
