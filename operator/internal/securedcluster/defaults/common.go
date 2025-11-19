package defaults

import (
	"reflect"

	"github.com/stackrox/rox/operator/api/v1alpha1"
)

// securedClusterStatusUninitialized checks if the provided SecuredClusterStatus is uninitialized.
func securedClusterStatusUninitialized(status *v1alpha1.SecuredClusterStatus) bool {
	return status == nil || reflect.DeepEqual(status, &v1alpha1.SecuredClusterStatus{})
}
