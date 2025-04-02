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
	"github.com/stackrox/rox/pkg/queue"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/certrepo"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/securedcluster"
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

	securedClusterComponentName    = "secured cluster"
	securedClusterSensorCapability = centralsensor.SecuredClusterCertificatesRefresh
	securedClusterResponseFn       = func(msg *central.MsgToSensor) *Response {
		return NewResponseFromSecuredClusterCerts(msg.GetIssueSecuredClusterCertsResponse())
	}
)

// NewSecuredClusterTLSIssuer creates a Sensor component that will keep the Secured Cluster certificates up to date
func NewSecuredClusterTLSIssuer(
	k8sClient kubernetes.Interface,
	sensorNamespace string,
	sensorPodName string,
) common.SensorComponent {
	return &tlsIssuerImpl{
		componentName:                securedClusterComponentName,
		sensorCapability:             securedClusterSensorCapability,
		getResponseFn:                securedClusterResponseFn,
		sensorNamespace:              sensorNamespace,
		sensorPodName:                sensorPodName,
		k8sClient:                    k8sClient,
		certRefreshBackoff:           certRefreshBackoff,
		getCertificateRefresherFn:    newCertificatesRefresher,
		getServiceCertificatesRepoFn: securedcluster.NewServiceCertificatesRepo,
		msgToCentralC:                make(chan *message.ExpiringMessage),
		newMsgFromSensorFn:           newSecuredClusterMsgFromSensor,
		responseQueue:                queue.NewQueue[*Response](),
		requiredCentralCapability: func() *centralsensor.CentralCapability {
			centralCap := centralsensor.CentralCapability(centralsensor.SecuredClusterCertificatesReissue)
			return &centralCap
		}(),
	}
}

func newSecuredClusterMsgFromSensor(requestID string) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_IssueSecuredClusterCertsRequest{
			IssueSecuredClusterCertsRequest: &central.IssueSecuredClusterCertsRequest{
				RequestId: requestID,
			},
		},
	}
}

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
	responseQueue                *queue.Queue[*Response]
	requestMutex                 sync.Mutex
	newMsgFromSensorFn           newMsgFromSensor
	requiredCentralCapability    *centralsensor.CentralCapability
	started                      atomic.Bool
	online                       atomic.Bool
	cancelRefresher              context.CancelFunc
	activateLock                 sync.Mutex
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
	i.activateLock.Lock()
	defer i.activateLock.Unlock()

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
	i.activateLock.Lock()
	defer i.activateLock.Unlock()

	if i.certRefresher != nil {
		i.cancelRefresher()
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

	i.responseQueue.Push(response)

	return nil
}

// requestCertificates makes a new request for a new set of secured cluster certificates from Central.
// Concurrent requests are *not* supported, this function will block until previous calls finish.
func (i *tlsIssuerImpl) requestCertificates(ctx context.Context) (*Response, error) {
	if i.requiredCentralCapability != nil {
		// Central capabilities are only available after this component is created,
		// which is why this check is done here
		if !centralcaps.Has(*i.requiredCentralCapability) {
			return nil, fmt.Errorf("TLS certificate refresh failed: missing Central capability %q", *i.requiredCentralCapability)
		}
	}

	// Ensure only one request can be active at a time. By protecting both send and receive
	// with the same mutex, we prevent a future send from starting while an older receive is still running.
	return concurrency.WithLock2(&i.requestMutex, func() (*Response, error) {
		requestID := uuid.NewV4().String()
		if err := i.send(ctx, requestID); err != nil {
			return nil, err
		}
		return i.receive(ctx, requestID)
	})
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
func (i *tlsIssuerImpl) receive(ctx context.Context, requestID string) (*Response, error) {
	for {
		response := i.responseQueue.PullBlocking(ctx)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if response == nil {
			return nil, errors.New("received nil response")
		}

		if response.RequestId == requestID {
			return response, nil
		}

		log.Warnf("Ignoring response, ID %q does not match ongoing request ID %q.",
			response.RequestId, requestID)
	}
}
