package extensions

import (
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
)

var (
	nonEmptySecuredClusterStatus = platform.SecuredClusterStatus{
		DeployedRelease: &platform.StackRoxRelease{
			Version: "some-version-string",
		},
	}
)
