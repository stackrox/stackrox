// Package operator contains "operational logic" so Sensor is able to operate itself when it is
// not deployed by our operator
package operator

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

var (
	log = logging.LoggerForModule()
)

// Operator performs some operator logic on deployments types that are not managed by our operator,
// like Helm or YAML bundle deployments
type Operator interface {
	Initialize(ctx context.Context) error
	Start(ctx context.Context) error
	Stopped() concurrency.ReadOnlyErrorSignal
	GetHelmReleaseRevision() uint64
}

type operatorImpl struct {
	k8sClient    kubernetes.Interface
	appNamespace string
	// Zero value if not managed by Helm
	helmReleaseName string
	// Zero value if not managed by Helm
	helmReleaseRevision uint64
	stoppedC            concurrency.ErrorSignal
}

// New creates a new operator
func New(k8sClient kubernetes.Interface, appNamespace string) Operator {
	return &operatorImpl{
		k8sClient:    k8sClient,
		appNamespace: appNamespace,
		stoppedC:     concurrency.NewErrorSignal(),
	}
}

func (o *operatorImpl) Initialize(ctx context.Context) error {
	log.Infof("Initializing operator for namespace %s", o.appNamespace)
	if err := o.fetchHelmReleaseName(ctx); err != nil {
		return o.failInitialization(err)
	}

	if err := o.fetchCurrentSensorHelmReleaseRevision(ctx); err != nil {
		return o.failInitialization(err)
	}

	return nil
}

func (o *operatorImpl) failInitialization(err error) error {
	return errors.Wrap(err, "Operator initialization error")
}

func (o *operatorImpl) GetHelmReleaseRevision() uint64 {
	return o.helmReleaseRevision
}

func (o *operatorImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return &o.stoppedC
}

func (o *operatorImpl) stop(err error) {
	o.stoppedC.SignalWithError(err)
}

// Start launches the processes that implement the "operational logic" of Sensor.
// Precondition: Initialize was previously invoked.
func (o *operatorImpl) Start(ctx context.Context) error {
	log.Info("Starting embedded operator.")

	if !o.isSensorHelmManaged() {
		log.Warn("Sensor is not managed by Helm, stopping the embedded operator as it only supports Helm.")
		return nil
	}

	// The current functionality to watch secrets is not critical, and it is very simple, so we disable resync
	// to avoid load in the k8s API. See https://groups.google.com/forum/#!topic/kubernetes-sig-api-machinery/PbSCXdLDno0
	// as linked in sensor/kubernetes/listener/listener_impl.go for context.
	noResyncPeriod := 0 * time.Minute
	sif := informers.NewSharedInformerFactoryWithOptions(o.k8sClient, noResyncPeriod, informers.WithNamespace(o.appNamespace))
	o.watchSecrets(sif)

	log.Info("Embedded operator started correctly.")
	return nil
}
