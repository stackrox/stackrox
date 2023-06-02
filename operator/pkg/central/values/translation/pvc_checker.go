package translation

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// pvcExistenceChecker wraps up the information and function to detect
type pvcExistenceChecker struct {
	ctx         context.Context
	client      ctrlClient.Client
	obsoletePvc bool
	nameSpace   string
}

func (c *pvcExistenceChecker) toObsolete() bool {
	return c.obsoletePvc
}

func (c *pvcExistenceChecker) pvcExists(name string) bool {
	key := ctrlClient.ObjectKey{Namespace: c.nameSpace, Name: name}
	pvc := &corev1.PersistentVolumeClaim{}
	err := c.client.Get(c.ctx, key, pvc)
	if !apiErrors.IsNotFound(err) {
		utils.Should(errors.Wrapf(err, "failed to check pvc %s in name space %s", name, c.nameSpace))
	}
	// In case of error, we do not know if there is exising pvc there. It would be safer to
	// assume it is not there. In that case, we may leave two persistent files not migrated
	// for offline mode. I am not sure how many customer working with operator in offline mode in the first place,
	// but that scenario can be corrected by upload them again.
	return err == nil
}
