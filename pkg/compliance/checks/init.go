package checks

import (
	// Make sure all checks from all standards are registered.
	_ "github.com/stackrox/rox/pkg/compliance/checks/hipaa_164"
	_ "github.com/stackrox/rox/pkg/compliance/checks/kubernetes"
	_ "github.com/stackrox/rox/pkg/compliance/checks/nist800-190"
	_ "github.com/stackrox/rox/pkg/compliance/checks/nist80053"
	_ "github.com/stackrox/rox/pkg/compliance/checks/pcidss32"
)

// Init registers all compliance checks.
// Called explicitly from central/app/app.go instead of package init().
// The actual registration happens in init() functions within each standard package.
func Init() {
	// The blank imports above ensure all standard packages are imported,
	// which triggers their init() functions that register the checks.
	// This function intentionally does nothing - it just needs to be called
	// to ensure the package (and its imports) are loaded.
}
