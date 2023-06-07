package translation

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// pvcStateChecker wraps up the information and function to determine pvc state
type pvcStateChecker struct {
	ctx       context.Context
	client    ctrlClient.Client
	namespace string
}

func (c *pvcStateChecker) pvcExists(name string) (bool, error) {
	key := ctrlClient.ObjectKey{Namespace: c.namespace, Name: name}
	pvc := &corev1.PersistentVolumeClaim{}
	if err := c.client.Get(c.ctx, key, pvc); err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}
		return false, errors.Wrap(err, "failed to get PVC central-db status")
	}
	return true, nil
}
