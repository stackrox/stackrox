package checks

import (
	// Make sure all checks from all standards are registered.
	_ "github.com/stackrox/rox/central/compliance/checks/hipaa_164"
	_ "github.com/stackrox/rox/central/compliance/checks/nist800-190"
	_ "github.com/stackrox/rox/central/compliance/checks/nist80053"
	_ "github.com/stackrox/rox/central/compliance/checks/pcidss32"
	_ "github.com/stackrox/rox/central/compliance/checks/remote"
)
