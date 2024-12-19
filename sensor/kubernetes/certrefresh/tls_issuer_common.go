package certrefresh

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/certificates"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/certrepo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	log                        = logging.LoggerForModule()
	_   common.SensorComponent = (*tlsIssuerImpl)(nil)

	startTimeout                         = 6 * time.Minute
	fetchSensorDeploymentOwnerRefBackoff = wait.Backoff{
		Duration: 10 * time.Millisecond,
		Factor:   3,
		Jitter:   0.1,
		Steps:    10,
		Cap:      startTimeout,
	}
	certRefreshTimeout = 5 * time.Minute
	certRefreshBackoff = wait.Backoff{
		Duration: 5 * time.Second,
		Factor:   3.0,
		Jitter:   0.1,
		Steps:    5,
		Cap:      10 * time.Minute,
	}
)

type certificateRefresherGetter func(certsDescription string, requestCertificates requestCertificatesFunc,
	repository certrepo.ServiceCertificatesRepo, timeout time.Duration, backoff wait.Backoff) concurrency.RetryTicker

type serviceCertificatesRepoGetter func(ownerReference metav1.OwnerReference, namespace string,
	secretsClient corev1.SecretInterface) certrepo.ServiceCertificatesRepo

type tlsIssuerImpl struct {
	componentName                string
	sensorCapability             centralsensor.SensorCapability
	getResponseFn                func(*central.MsgToSensor) *certificates.Response
	sensorNamespace              string
	sensorPodName                string
	k8sClient                    kubernetes.Interface
	certRefreshBackoff           wait.Backoff
	getCertificateRefresherFn    certificateRefresherGetter
	getServiceCertificatesRepoFn serviceCertificatesRepoGetter
	certRequester                certificates.Requester
	certRefresher                concurrency.RetryTicker
	msgToCentralC                chan *message.ExpiringMessage
	stopSig                      concurrency.ErrorSignal
}

// Start starts the Sensor component and launches a certificate refresher that immediately checks the certificates, and
// that keeps them updated.
// In case a secret doesn't have the expected owner, this logs a warning and returns nil.
// In case this component was already started it fails immediately.
func (i *tlsIssuerImpl) Start() error {
	log.Debugf("Starting %s TLS issuer.", i.componentName)
	i.stopSig.Reset()
	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()

	if i.certRefresher != nil {
		return i.abortStart(errors.New("already started"))
	}

	sensorOwnerReference, fetchSensorDeploymentErr := FetchSensorDeploymentOwnerRef(ctx, i.sensorPodName,
		i.sensorNamespace, i.k8sClient, fetchSensorDeploymentOwnerRefBackoff)
	if fetchSensorDeploymentErr != nil {
		return i.abortStart(errors.Wrap(fetchSensorDeploymentErr, "fetching sensor deployment"))
	}

	certsRepo := i.getServiceCertificatesRepoFn(*sensorOwnerReference, i.sensorNamespace,
		i.k8sClient.CoreV1().Secrets(i.sensorNamespace))
	i.certRefresher = i.getCertificateRefresherFn(i.componentName, i.certRequester.RequestCertificates, certsRepo,
		certRefreshTimeout, i.certRefreshBackoff)

	if refreshStartErr := i.certRefresher.Start(); refreshStartErr != nil {
		return i.abortStart(errors.Wrap(refreshStartErr, "starting certificate refresher"))
	}

	log.Debugf("%s TLS issuer started.", i.componentName)
	return nil
}

func (i *tlsIssuerImpl) abortStart(err error) error {
	log.Errorf("%s TLS issuer start aborted due to error: %s", i.componentName, err)
	i.Stop(err)
	return err
}

func (i *tlsIssuerImpl) Stop(_ error) {
	i.stopSig.Signal()
	if i.certRefresher != nil {
		i.certRefresher.Stop()
		i.certRefresher = nil
	}

	log.Debugf("%s TLS issuer stopped.", i.componentName)
}

func (i *tlsIssuerImpl) Notify(common.SensorComponentEvent) {}

func (i *tlsIssuerImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{i.sensorCapability}
}

// ResponsesC is called "responses" because for other SensorComponent it is Central that
// initiates the interaction. However, here it is Sensor which sends a request to Central.
func (i *tlsIssuerImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return i.msgToCentralC
}

// ProcessMessage dispatches Central's messages to Sensor received via the Central receiver.
// This method must not block as it would prevent centralReceiverImpl from sending messages
// to other SensorComponents.
func (i *tlsIssuerImpl) ProcessMessage(msg *central.MsgToSensor) error {
	response := i.getResponseFn(msg)
	if response == nil {
		// messages not supported by this component are ignored
		return nil
	}

	go func() {
		i.certRequester.DispatchResponse(response)
	}()

	return nil
}

func (i *tlsIssuerImpl) msgToCentralHandler(ctx context.Context, msg *message.ExpiringMessage) error {
	select {
	case <-i.stopSig.Done():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case i.msgToCentralC <- msg:
		return nil
	}
}
