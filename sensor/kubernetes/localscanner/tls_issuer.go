package localscanner

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	appsApiv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	log = logging.LoggerForModule()

	scannerSpec = ServiceCertSecretSpec{
		secretName:          "scanner-slim-tls",
		caCertFileName:      "ca.pem", // FIXME review this and for db
		serviceCertFileName: "cert.pem",
		serviceKeyFileName:  "key.pem",
	}
	scannerDBSpec = ServiceCertSecretSpec{
		secretName:          "scanner-slim-db-tls",
		caCertFileName:      "ca.pem",
		serviceCertFileName: "cert.pem",
		serviceKeyFileName:  "key.pem",
	}

	startTimeout          = 5 * time.Minute
	processMessageTimeout = time.Second
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
		certificateRefresherSupplier:    newCertRefresher,
		serviceCertificatesRepoSupplier: NewServiceCertificatesRepo,
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
	refresher                       CertificateRefresher
}

// CertificateRequester requests a new set of local scanner certificates from central.
type CertificateRequester interface {
	Start()
	Stop()
	RequestCertificates(ctx context.Context) (*central.IssueLocalScannerCertsResponse, error)
}

// CertificateRefresher periodically checks the expiration of a set of certificates, and if needed
// requests new certificates to central and updates those certificates.
type CertificateRefresher interface {
	Start() error
	Stop()
}

type certificateRefresherSupplier func(requestCertificates requestCertificatesFunc, timeout time.Duration,
	backoff wait.Backoff, repository ServiceCertificatesRepo) CertificateRefresher

// ServiceCertificatesRepo TODO replace by ROX-9148
type ServiceCertificatesRepo interface{}

// requestCertificatesFunc TODO replace by ROX-9148
type requestCertificatesFunc func(ctx context.Context) (*central.IssueLocalScannerCertsResponse, error)

// newCertRefresher TODO replace by ROX-9148
func newCertRefresher(requestCertificates requestCertificatesFunc, timeout time.Duration,
	backoff wait.Backoff, repository ServiceCertificatesRepo) CertificateRefresher {
	return nil
}

// ServiceCertSecretSpec TODO replace by ROX-9128
type ServiceCertSecretSpec struct {
	secretName          string
	caCertFileName      string
	serviceCertFileName string
	serviceKeyFileName  string
}

// NewServiceCertificatesRepo TODO replace by ROX-9128
func NewServiceCertificatesRepo(ctx context.Context, scannerSpec, scannerDBSpec ServiceCertSecretSpec,
	sensorDeployment *appsApiv1.Deployment,
	initialCertsSupplier func(context.Context) (*storage.TypedServiceCertificateSet, error),
	secretsClient corev1.SecretInterface) (ServiceCertificatesRepo, error) {
	return nil, nil
}

type serviceCertificatesRepoSupplier func(ctx context.Context, scannerSpec, scannerDBSpec ServiceCertSecretSpec,
	sensorDeployment *appsApiv1.Deployment,
	initialCertsSupplier func(context.Context) (*storage.TypedServiceCertificateSet, error),
	secretsClient corev1.SecretInterface) (ServiceCertificatesRepo, error)

func (i *localScannerTLSIssuerImpl) Start() error {
	log.Info("starting local scanner TLS issuer.")
	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()

	if i.refresher != nil {
		return errors.New("already started")
	}

	sensorDeployment, getSensorDeploymentErr := i.fetchSensorDeployment(ctx)
	if getSensorDeploymentErr != nil {
		return errors.Wrap(getSensorDeploymentErr, "fetching sensor deployment")
	}

	i.requester.Start()

	certsRepo, createCertsRepoErr := i.serviceCertificatesRepoSupplier(ctx, scannerSpec, scannerDBSpec, sensorDeployment,
		i.initialCertsSupplier(), i.k8sClient.CoreV1().Secrets(i.sensorNamespace))
	if createCertsRepoErr != nil {
		return errors.Wrap(createCertsRepoErr, "creating service certificates repository")
	}
	i.refresher = i.certificateRefresherSupplier(i.requester.RequestCertificates, certRefreshTimeout, certRefreshBackoff, certsRepo)
	if refreshStartErr := i.refresher.Start(); refreshStartErr != nil {
		return errors.Wrap(refreshStartErr, "starting certificate refresher")
	}

	log.Info("local scanner TLS issuer started.")

	return nil
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

// initialCertsSupplier request the certificates at most once, and returns the memoized response to that single request.
func (i *localScannerTLSIssuerImpl) initialCertsSupplier() func(context.Context) (*storage.TypedServiceCertificateSet, error) {
	var (
		requestOnce  sync.Once
		certificates *storage.TypedServiceCertificateSet
		requestErr   error
	)
	requestCertificates := i.requester.RequestCertificates
	return func(ctx context.Context) (*storage.TypedServiceCertificateSet, error) {
		requestOnce.Do(func() {
			response, err := requestCertificates(ctx)
			if err != nil {
				requestErr = err
				return
			}
			if response.GetError() != nil {
				requestErr = errors.Errorf("central refused to issue certificates: %s", response.GetError().GetMessage())
				return
			}
			certificates = response.GetCertificates()
		})
		return certificates, requestErr
	}
}

func (i *localScannerTLSIssuerImpl) fetchSensorDeployment(ctx context.Context) (*appsApiv1.Deployment, error) {
	if i.podOwnerName == "" {
		return nil, errors.New("fetching sensor deployment: empty pod owner name")
	}

	replicaSetClient := i.k8sClient.AppsV1().ReplicaSets(i.sensorNamespace)
	ownerReplicaSet, getReplicaSetErr := replicaSetClient.Get(ctx, i.podOwnerName, metav1.GetOptions{})
	if getReplicaSetErr != nil {
		return nil, errors.Wrap(getReplicaSetErr, "fetching owner replica set")
	}

	replicaSetOwners := ownerReplicaSet.GetObjectMeta().GetOwnerReferences()
	if len(replicaSetOwners) != 1 {
		return nil, errors.Errorf("fetching sensor deployment: replica set %q has unexpected owners %v",
			ownerReplicaSet.GetName(), replicaSetOwners)
	}
	replicaSetOwner := replicaSetOwners[0]

	deploymentClient := i.k8sClient.AppsV1().Deployments(i.sensorNamespace)
	sensorDeployment, getSensorDeploymentErr := deploymentClient.Get(ctx, replicaSetOwner.Name, metav1.GetOptions{})
	if getSensorDeploymentErr != nil {
		return nil, errors.Wrap(getReplicaSetErr, "fetching sensor deployment")
	}

	return sensorDeployment, nil
}
