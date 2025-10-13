package checks

import (
	// Make sure all checks from all standards are registered.
	_ "github.com/stackrox/rox/pkg/compliance/checks/hipaa_164"
	_ "github.com/stackrox/rox/pkg/compliance/checks/kubernetes"
	_ "github.com/stackrox/rox/pkg/compliance/checks/nist800-190"
	_ "github.com/stackrox/rox/pkg/compliance/checks/nist80053"
	_ "github.com/stackrox/rox/pkg/compliance/checks/pcidss32"
)
