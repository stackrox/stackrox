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
	appsApiv1 "k8s.io/api/apps/v1"
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
func NewLocalScannerTLSIssuer(k8sClient kubernetes.Interface, sensorNamespace string,
	podOwnerName string) common.SensorComponent {
	msgToCentralC := make(chan *central.MsgFromSensor)
	msgFromCentralC := make(chan *central.IssueLocalScannerCertsResponse)
	return &localScannerTLSIssuerImpl{
		sensorNamespace:                 sensorNamespace,
		podOwnerName:                    podOwnerName,
		k8sClient:                       k8sClient,
		msgToCentralC:                   msgToCentralC,
		msgFromCentralC:                 msgFromCentralC,
		certificateRefresherSupplier:    newCertificatesRefresher,
		serviceCertificatesRepoSupplier: newServiceCertificatesRepo,
		requester:                       NewCertificateRequester(msgToCentralC, msgFromCentralC),
	}
}

type localScannerTLSIssuerImpl struct {
	sensorNamespace                 string
	podOwnerName                    string
	k8sClient                       kubernetes.Interface
	msgToCentralC                   chan *central.MsgFromSensor
	msgFromCentralC                 chan *central.IssueLocalScannerCertsResponse
	certificateRefresherSupplier    certificateRefresherSupplier
	serviceCertificatesRepoSupplier serviceCertificatesRepoSupplier
	requester                       CertificateRequester
	refresher                       concurrency.RetryTicker
}

// CertificateRequester requests a new set of local scanner certificates from central.
type CertificateRequester interface {
	Start()
	Stop()
	RequestCertificates(ctx context.Context) (*central.IssueLocalScannerCertsResponse, error)
}

type certificateRefresherSupplier func(requestCertificates requestCertificatesFunc, repository serviceCertificatesRepo,
	timeout time.Duration, backoff wait.Backoff) concurrency.RetryTicker

type serviceCertificatesRepoSupplier func(ownerReference metav1.OwnerReference, namespace string,
	secretsClient corev1.SecretInterface) serviceCertificatesRepo

// Start launches a certificate refreshes that immediately checks the certificates, and that keeps them updated.
// In case a secret doesn't have the expected owner, this logs a warning and returns nil.
// In case this component was already started it fails immediately.
func (i *localScannerTLSIssuerImpl) Start() error {
	log.Info("starting local scanner TLS issuer.")
	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()

	if i.refresher != nil {
		return i.abortStart(errors.New("already started"))
	}

	sensorOwnerReference, fetchSensorDeploymentErr := i.fetchSensorDeploymentOwnerRef(ctx, fetchSensorDeploymentOwnerRefBackoff)
	if fetchSensorDeploymentErr != nil {
		return i.abortStart(errors.Wrap(fetchSensorDeploymentErr, "fetching sensor deployment"))
	}

	certsRepo := i.serviceCertificatesRepoSupplier(*sensorOwnerReference, i.sensorNamespace,
		i.k8sClient.CoreV1().Secrets(i.sensorNamespace))
	i.refresher = i.certificateRefresherSupplier(i.requester.RequestCertificates, certsRepo,
		certRefreshTimeout, certRefreshBackoff)

	i.requester.Start()
	if refreshStartErr := i.refresher.Start(); refreshStartErr != nil {
		return i.abortStart(errors.Wrap(refreshStartErr, "starting certificate refresher"))
	}

	log.Info("local scanner TLS issuer started.")
	return nil
}

func (i *localScannerTLSIssuerImpl) abortStart(err error) error {
	log.Errorf("local scanner TLS issuer start aborted due to error: %s", err)
	i.Stop(err)
	return err
}

func (i *localScannerTLSIssuerImpl) Stop(err error) {
	if i.refresher != nil {
		i.refresher.Stop()
		i.refresher = nil
	}

	i.requester.Stop()
	log.Info("local scanner TLS issuer stopped.")
}

func (i *localScannerTLSIssuerImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.LocalScannerCredentialsRefresh}
}

// ResponsesC is called "responses" because for other SensorComponent it is central that
// initiates the interaction. However, here it is sensor which sends a request to central.
func (i *localScannerTLSIssuerImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return i.msgToCentralC
}

// ProcessMessage is how the central receiver delivers messages from central to SensorComponents.
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
				// refresher will retry.
				log.Errorf("timeout forwarding response %s from central: %s", response, ctx.Err())
			case i.msgFromCentralC <- response:
			}
		}()
		return nil
	default:
		// silently ignore other messages broadcasted by centralReceiverImpl, as centralReceiverImpl logs
		// all returned errors with error level.
		return nil
	}
}

func (i *localScannerTLSIssuerImpl) fetchSensorDeploymentOwnerRef(ctx context.Context,
	backoff wait.Backoff) (*metav1.OwnerReference, error) {

	if i.podOwnerName == "" {
		return nil, errors.New("fetching sensor deployment: empty pod owner name")
	}

	replicaSetClient := i.k8sClient.AppsV1().ReplicaSets(i.sensorNamespace)
	var ownerReplicaSet *appsApiv1.ReplicaSet
	getReplicaSetErr := retry.OnError(backoff, func(err error) bool {
		return !k8sErrors.IsNotFound(err)
	}, func() error {
		replicaSet, getReplicaSetErr := replicaSetClient.Get(ctx, i.podOwnerName, metav1.GetOptions{})
		ownerReplicaSet = replicaSet
		return getReplicaSetErr
	})
	if getReplicaSetErr != nil {
		return nil, errors.Wrap(getReplicaSetErr, "fetching owner replica set")
	}

	replicaSetOwners := ownerReplicaSet.GetObjectMeta().GetOwnerReferences()
	if len(replicaSetOwners) != 1 {
		return nil, errors.Errorf("fetching sensor deployment: replica set %q has unexpected owners %v",
			ownerReplicaSet.GetName(), replicaSetOwners)
	}
	replicaSetOwner := replicaSetOwners[0]
	blockOwnerDeletion := false
	isController := false
	return &metav1.OwnerReference{
		APIVersion:         appsApiv1.SchemeGroupVersion.String(),
		Kind:               "Deployment",
		Name:               replicaSetOwner.Name,
		UID:                replicaSetOwner.UID,
		BlockOwnerDeletion: &blockOwnerDeletion,
		Controller:         &isController,
	}, nil
}
