package certrefresh

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/certrepo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	log                        = logging.LoggerForModule()
	_   common.SensorComponent = (*tlsIssuerImpl)(nil)

	startTimeout       = 6 * time.Minute
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

type newMsgFromSensor func(requestID string) *central.MsgFromSensor

type tlsIssuerImpl struct {
	componentName                string
	sensorCapability             centralsensor.SensorCapability
	getResponseFn                func(*central.MsgToSensor) *Response
	sensorNamespace              string
	sensorPodName                string
	k8sClient                    kubernetes.Interface
	certRefreshBackoff           wait.Backoff
	getCertificateRefresherFn    certificateRefresherGetter
	getServiceCertificatesRepoFn serviceCertificatesRepoGetter
	certRefresher                concurrency.RetryTicker
	msgToCentralC                chan *message.ExpiringMessage
	stopSig                      concurrency.ErrorSignal
	responseFromCentral          atomic.Pointer[Response]
	responseReceived             concurrency.Signal
	ongoingRequestID             string
	ongoingRequestIDMutex        sync.Mutex
	requestOngoing               atomic.Bool
	newMsgFromSensorFn           newMsgFromSensor
	requiredCentralCapability    *centralsensor.CentralCapability
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
		i.sensorNamespace, i.k8sClient, wait.Backoff{})
	if fetchSensorDeploymentErr != nil {
		return i.abortStart(errors.Wrap(fetchSensorDeploymentErr, "fetching sensor deployment"))
	}

	certsRepo := i.getServiceCertificatesRepoFn(*sensorOwnerReference, i.sensorNamespace,
		i.k8sClient.CoreV1().Secrets(i.sensorNamespace))
	i.certRefresher = i.getCertificateRefresherFn(i.componentName, i.requestCertificates, certsRepo,
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
	if i.getResponseFn == nil {
		return errors.New("getResponseFn is not set")
	}

	response := i.getResponseFn(msg)
	if response == nil {
		// messages not supported by this component are ignored
		return nil
	}

	go i.dispatch(response)
	return nil
}

func (i *tlsIssuerImpl) dispatch(response *Response) {
	if !i.requestOngoing.Load() {
		log.Warnf("Received a response for request ID %q, but there is no ongoing RequestCertificates call.", response.RequestId)
		return
	}
	ongoingRequestID := concurrency.WithLock1(&i.ongoingRequestIDMutex, func() string {
		return i.ongoingRequestID
	})
	if ongoingRequestID != response.RequestId {
		log.Warnf("Request ID %q does not match ongoing request ID %q.",
			response.RequestId, ongoingRequestID)
		return
	}

	i.responseFromCentral.Store(response)
	i.responseReceived.Signal()
}

// requestCertificates makes a new request for a new set of secured cluster certificates from Central.
// Concurrent requests are *not* supported.
func (i *tlsIssuerImpl) requestCertificates(ctx context.Context) (*Response, error) {
	if i.requiredCentralCapability != nil {
		// Central capabilities are only available after this component is created,
		// which is why this check is done here
		if !centralcaps.Has(*i.requiredCentralCapability) {
			return nil, fmt.Errorf("TLS certificate refresh failed: missing Central capability %q", *i.requiredCentralCapability)
		}
	}

	if !i.requestOngoing.CompareAndSwap(false, true) {
		return nil, errors.New("concurrent requests are not supported.")
	}

	defer func() {
		i.requestOngoing.Store(false)
		i.responseReceived.Reset()
	}()

	requestID := uuid.NewV4().String()
	concurrency.WithLock(&i.ongoingRequestIDMutex, func() {
		i.ongoingRequestID = requestID
	})

	if err := i.send(ctx, requestID); err != nil {
		return nil, err
	}
	return i.receive(ctx)
}

// send a cert refresh request to Central
func (i *tlsIssuerImpl) send(ctx context.Context, requestID string) error {
	select {
	case <-i.stopSig.Done():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case i.msgToCentralC <- message.New(i.newMsgFromSensorFn(requestID)):
		return nil
	}
}

// receive handles the response to a specific certificate request
func (i *tlsIssuerImpl) receive(ctx context.Context) (*Response, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-i.responseReceived.Done():
		return i.responseFromCentral.Load(), nil
	}
}
