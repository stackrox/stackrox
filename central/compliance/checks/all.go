package checks

import (
	// Make sure all checks from all standards are registered.
	_ "github.com/stackrox/rox/central/compliance/checks/docker"
	_ "github.com/stackrox/rox/central/compliance/checks/hipaa_164"
	_ "github.com/stackrox/rox/central/compliance/checks/kubernetes"
	_ "github.com/stackrox/rox/central/compliance/checks/nist800-190"
	_ "github.com/stackrox/rox/central/compliance/checks/pcidss32"
)
