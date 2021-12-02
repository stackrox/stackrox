// Package operator contains "operational logic" so Sensor is able to operate itself when it is
// not deployed by our operator
package operator

import (
	"context"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	log = logging.LoggerForModule()
)

type Operator interface {
	Start(ctx context.Context) error
}

type operatorImpl struct {
	k8sClient kubernetes.Interface
	appNamespace string
	// Zero value if not managed by Helm
	helmReleaseName string
	// Zero value if not managed by Helm
	helmReleaseRevision uint64
}

func newOperator(k8sClient kubernetes.Interface, appNamespace string, pod v1.Pod) Operator {
	return &operatorImpl{
		k8sClient: k8sClient,
		appNamespace: appNamespace,
		helmReleaseName: getHelmReleaseName(pod),
	}
}

func (o *operatorImpl) Start(ctx context.Context) error {
	// TODO: // Secret watch ... or informer ..
	//  // see https://pkg.go.dev/k8s.io/client-go/tools/watch#NewIndexerInformerWatcher
	// FIXME
	for { // FIXME proper loop
		var secret *v1.Secret
		err := o.processSecret(secret)
		if err != nil {
			err := errors.Wrapf(err, "Error processing secret with name %s", secret.GetName())
			log.Error(err)
		}

		break // FIXME just so it doesn't loop forever
	}

	// TODO: stop immediately if ! o.IsSensorHelmManaged()

	return nil // FIXME this should be the initialization error
}