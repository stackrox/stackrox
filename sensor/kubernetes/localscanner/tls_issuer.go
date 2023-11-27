package localscanner

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
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/util/retry"
)

var (
	log = logging.LoggerForModule()

	startTimeout                         = 6 * time.Minute
	fetchSensorDeploymentOwnerRefBackoff = wait.Backoff{
		Duration: 10 * time.Millisecond,
		Factor:   3,
		Jitter:   0.1,
		Steps:    10,
		Cap:      startTimeout,
	}
	processMessageTimeout = 5 * time.Second
	certRefreshTimeout    = 5 * time.Minute
	certRefreshBackoff    = wait.Backoff{
		Duration: 5 * time.Second,
		Factor:   3.0,
		Jitter:   0.1,
		Steps:    5,
		Cap:      10 * time.Minute,
	}
	_ common.SensorComponent = (*localScannerTLSIssuerImpl)(nil)
)

// NewLocalScannerTLSIssuer creates a sensor component that will keep the local scanner certificates
// up to date, using the specified retry parameters.
func NewLocalScannerTLSIssuer(
	k8sClient kubernetes.Interface,
	sensorNamespace string,
	sensorPodName string,
) common.SensorComponent {
	msgToCentralC := make(chan *message.ExpiringMessage)
	msgFromCentralC := make(chan *central.IssueLocalScannerCertsResponse)
	return &localScannerTLSIssuerImpl{
		sensorNamespace:              sensorNamespace,
		sensorPodName:                sensorPodName,
		k8sClient:                    k8sClient,
		msgToCentralC:                msgToCentralC,
		msgFromCentralC:              msgFromCentralC,
		certRefreshBackoff:           certRefreshBackoff,
		getCertificateRefresherFn:    newCertificatesRefresher,
		getServiceCertificatesRepoFn: newServiceCertificatesRepo,
		certRequester:                NewCertificateRequester(msgToCentralC, msgFromCentralC),
	}
}

type localScannerTLSIssuerImpl struct {
	sensorNamespace              string
	sensorPodName                string
	k8sClient                    kubernetes.Interface
	msgToCentralC                chan *message.ExpiringMessage
	msgFromCentralC              chan *central.IssueLocalScannerCertsResponse
	certRefreshBackoff           wait.Backoff
	getCertificateRefresherFn    certificateRefresherGetter
	getServiceCertificatesRepoFn serviceCertificatesRepoGetter
	certRequester                CertificateRequester
	certRefresher                concurrency.RetryTicker
}

// CertificateRequester requests a new set of local scanner certificates from central.
type CertificateRequester interface {
	Start()
	Stop()
	RequestCertificates(ctx context.Context) (*central.IssueLocalScannerCertsResponse, error)
}

type certificateRefresherGetter func(requestCertificates requestCertificatesFunc, repository serviceCertificatesRepo,
	timeout time.Duration, backoff wait.Backoff) concurrency.RetryTicker

type serviceCertificatesRepoGetter func(ownerReference metav1.OwnerReference, namespace string,
	secretsClient corev1.SecretInterface) serviceCertificatesRepo

// Start starts the sensor component and launches a certificate refresher that immediately checks the certificates, and
// that keeps them updated.
// In case a secret doesn't have the expected owner, this logs a warning and returns nil.
// In case this component was already started it fails immediately.
func (i *localScannerTLSIssuerImpl) Start() error {
	log.Debug("starting local scanner TLS issuer.")
	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()

	if i.certRefresher != nil {
		return i.abortStart(errors.New("already started"))
	}

	sensorOwnerReference, fetchSensorDeploymentErr := i.fetchSensorDeploymentOwnerRef(ctx, fetchSensorDeploymentOwnerRefBackoff)
	if fetchSensorDeploymentErr != nil {
		return i.abortStart(errors.Wrap(fetchSensorDeploymentErr, "fetching sensor deployment"))
	}

	certsRepo := i.getServiceCertificatesRepoFn(*sensorOwnerReference, i.sensorNamespace,
		i.k8sClient.CoreV1().Secrets(i.sensorNamespace))
	i.certRefresher = i.getCertificateRefresherFn(i.certRequester.RequestCertificates, certsRepo,
		certRefreshTimeout, i.certRefreshBackoff)

	i.certRequester.Start()
	if refreshStartErr := i.certRefresher.Start(); refreshStartErr != nil {
		return i.abortStart(errors.Wrap(refreshStartErr, "starting certificate certRefresher"))
	}

	log.Debug("local scanner TLS issuer started.")
	return nil
}

func (i *localScannerTLSIssuerImpl) abortStart(err error) error {
	log.Errorf("local scanner TLS issuer start aborted due to error: %s", err)
	i.Stop(err)
	return err
}

func (i *localScannerTLSIssuerImpl) Stop(_ error) {
	if i.certRefresher != nil {
		i.certRefresher.Stop()
		i.certRefresher = nil
	}

	i.certRequester.Stop()
	log.Debug("local scanner TLS issuer stopped.")
}

func (i *localScannerTLSIssuerImpl) Notify(common.SensorComponentEvent) {}

func (i *localScannerTLSIssuerImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.LocalScannerCredentialsRefresh}
}

// ResponsesC is called "responses" because for other SensorComponent it is central that
// initiates the interaction. However, here it is sensor which sends a request to central.
func (i *localScannerTLSIssuerImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return i.msgToCentralC
}

// ProcessMessage dispatches Central's messages to Sensor received via the central receiver.
// This method must not block as it would prevent centralReceiverImpl from sending messages
// to other SensorComponents.
func (i *localScannerTLSIssuerImpl) ProcessMessage(msg *central.MsgToSensor) error {
	switch m := msg.GetMsg().(type) {
	case *central.MsgToSensor_IssueLocalScannerCertsResponse:
		response := m.IssueLocalScannerCertsResponse
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), processMessageTimeout)
			defer cancel()
			select {
			case <-ctx.Done():
				// certRefresher will retry.
				log.Errorf("timeout forwarding response %s from central: %s", response, ctx.Err())
			case i.msgFromCentralC <- response:
			}
		}()
		return nil
	default:
		// messages not supported by this component are ignored because unknown messages types are handled by the central receiver.
		return nil
	}
}

func (i *localScannerTLSIssuerImpl) fetchSensorDeploymentOwnerRef(ctx context.Context, backoff wait.Backoff) (*metav1.OwnerReference, error) {
	if i.sensorPodName == "" {
		return nil, errors.New("fetching sensor deployment: empty pod name")
	}

	podsClient := i.k8sClient.CoreV1().Pods(i.sensorNamespace)
	sensorPodMeta, getPodErr := i.getObjectMetaWithRetries(ctx, backoff, func(ctx context.Context) (metav1.Object, error) {
		pod, err := podsClient.Get(ctx, i.sensorPodName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return pod.GetObjectMeta(), nil
	})
	if getPodErr != nil {
		return nil, errors.Wrapf(getPodErr, "fetching sensor pod with name %q", i.sensorPodName)
	}
	podOwners := sensorPodMeta.GetOwnerReferences()
	if len(podOwners) != 1 {
		return nil, errors.Errorf("pod %q has unexpected owners %v",
			i.sensorPodName, podOwners)
	}
	podOwnerName := podOwners[0].Name

	replicaSetClient := i.k8sClient.AppsV1().ReplicaSets(i.sensorNamespace)
	ownerReplicaSetMeta, getReplicaSetErr := i.getObjectMetaWithRetries(ctx, backoff,
		func(ctx context.Context) (metav1.Object, error) {
			replicaSet, err := replicaSetClient.Get(ctx, podOwnerName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return replicaSet.GetObjectMeta(), nil
		})
	if getReplicaSetErr != nil {
		return nil, errors.Wrapf(getReplicaSetErr, "fetching owner replica set with name %q", podOwnerName)
	}
	replicaSetOwners := ownerReplicaSetMeta.GetOwnerReferences()
	if len(replicaSetOwners) != 1 {
		return nil, errors.Errorf("replica set %q has unexpected owners %v",
			ownerReplicaSetMeta.GetName(),
			replicaSetOwners)
	}
	replicaSetOwner := replicaSetOwners[0]

	blockOwnerDeletion := false
	isController := false
	return &metav1.OwnerReference{
		APIVersion:         replicaSetOwner.APIVersion,
		Kind:               replicaSetOwner.Kind,
		Name:               replicaSetOwner.Name,
		UID:                replicaSetOwner.UID,
		BlockOwnerDeletion: &blockOwnerDeletion,
		Controller:         &isController,
	}, nil
}

func (i *localScannerTLSIssuerImpl) getObjectMetaWithRetries(
	ctx context.Context,
	backoff wait.Backoff,
	getObject func(context.Context) (metav1.Object, error),
) (metav1.Object, error) {
	var object metav1.Object
	getErr := retry.OnError(backoff, func(err error) bool {
		return !k8sErrors.IsNotFound(err)
	}, func() error {
		newObject, err := getObject(ctx)
		if err == nil {
			object = newObject
		}
		return err
	})

	return object, getErr
}
