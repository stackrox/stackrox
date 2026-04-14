package checks

import (
	hipaa164 "github.com/stackrox/rox/pkg/compliance/checks/hipaa_164"
	"github.com/stackrox/rox/pkg/compliance/checks/kubernetes"
	nist800190 "github.com/stackrox/rox/pkg/compliance/checks/nist800-190"
	"github.com/stackrox/rox/pkg/compliance/checks/nist80053"
	"github.com/stackrox/rox/pkg/compliance/checks/pcidss32"
)

// Init registers all compliance checks.
// Called explicitly from central/app/app.go instead of package init().
func Init() {
	hipaa164.Init()
	kubernetes.Init()
	nist800190.Init()
	nist80053.Init()
	pcidss32.Init()
}
