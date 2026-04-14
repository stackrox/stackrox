package checks

import (
	hipaa164 "github.com/stackrox/rox/central/compliance/checks/hipaa_164"
	nist800190 "github.com/stackrox/rox/central/compliance/checks/nist800-190"
	"github.com/stackrox/rox/central/compliance/checks/nist80053"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32"
	"github.com/stackrox/rox/central/compliance/checks/remote"
)

// Init registers all central compliance checks.
// Called explicitly from central/compliance/manager/checks.go instead of package init().
func Init() {
	hipaa164.Init()
	nist800190.Init()
	nist80053.Init()
	pcidss32.Init()
	remote.Init()
}
