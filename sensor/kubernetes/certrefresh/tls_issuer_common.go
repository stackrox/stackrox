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
	responseFromCentral          atomic.Pointer[Response]
	responseReceived             concurrency.Signal
	ongoingRequestID             string
	ongoingRequestIDMutex        sync.Mutex
	requestOngoing               atomic.Bool
	newMsgFromSensorFn           newMsgFromSensor
	requiredCentralCapability    *centralsensor.CentralCapability
	started                      atomic.Bool
	online                       atomic.Bool
	cancelRefresher              context.CancelFunc
}

// Start starts the Sensor component and launches a certificate refresher that:
// * checks the state of the certificates whenever Sensor connects to Central, and several months before they expire
// * updates the certificates if needed
// When Sensor is offline this component is not active.
func (i *tlsIssuerImpl) Start() error {
	log.Debugf("Starting %s TLS issuer.", i.componentName)
	i.started.Store(true)
	return i.activate()
}

func (i *tlsIssuerImpl) activate() error {
	if !i.started.Load() {
		return nil
	}
	if !i.online.Load() {
		return nil
	}
	if i.certRefresher != nil {
		log.Debugf("%s TLS issuer is already started.", i.componentName)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()

	sensorOwnerReference, fetchSensorDeploymentErr := FetchSensorDeploymentOwnerRef(ctx, i.sensorPodName,
		i.sensorNamespace, i.k8sClient, wait.Backoff{})
	if fetchSensorDeploymentErr != nil {
		i.started.Store(false)
		return fmt.Errorf("fetching sensor deployment: %w", fetchSensorDeploymentErr)
	}

	certsRepo := i.getServiceCertificatesRepoFn(*sensorOwnerReference, i.sensorNamespace,
		i.k8sClient.CoreV1().Secrets(i.sensorNamespace))
	i.certRefresher = i.getCertificateRefresherFn(i.componentName, i.requestCertificates, certsRepo,
		certRefreshTimeout, i.certRefreshBackoff)

	refresherCtx, cancelFunc := context.WithCancel(context.Background())
	i.cancelRefresher = cancelFunc
	if refreshStartErr := i.certRefresher.Start(refresherCtx); refreshStartErr != nil {
		// Starting a RetryTicker should only return an error if already started or stopped, so this should
		// never happen because i.certRefresher was just created
		i.started.Store(false)
		return fmt.Errorf("starting certificate refresher: %w", refreshStartErr)
	}

	log.Debugf("%s TLS issuer is active.", i.componentName)
	return nil
}

func (i *tlsIssuerImpl) Stop(_ error) {
	i.started.Store(false)
	i.deactivate()
}

func (i *tlsIssuerImpl) deactivate() {
	i.cancelRefresher()
	if i.certRefresher != nil {
		i.certRefresher.Stop()
		i.certRefresher = nil
	}

	log.Debugf("%s TLS issuer is not active.", i.componentName)
}

func (i *tlsIssuerImpl) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e))

	switch e {
	case common.SensorComponentEventCentralReachable:
		// At this point we can be sure that Central capabilities have been received
		if i.requiredCentralCapability != nil && !centralcaps.Has(*i.requiredCentralCapability) {
			log.Infof("Central does not have the %s capability", i.requiredCentralCapability.String())
			return
		}
		i.online.Store(true)
		if err := i.activate(); err != nil {
			log.Warnf("Failed to activate %s TLS issuer: %v", i.componentName, err)
		}
	case common.SensorComponentEventOfflineMode:
		i.online.Store(false)
		i.deactivate()
	}
}

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
