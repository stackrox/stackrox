package defaults

import (
	"reflect"

	platform "github.com/stackrox/rox/operator/api/v1alpha1"
)

// isNewInstallation checks if this is a new installation based on the status.
func isNewInstallation(status *platform.SecuredClusterStatus) bool {
	// The ProductVersion is only set post installation.
	return status == nil ||
		reflect.DeepEqual(status, &platform.SecuredClusterStatus{}) ||
		status.ProductVersion == ""
}
