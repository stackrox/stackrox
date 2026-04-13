package hipaa164

import (
	"github.com/stackrox/rox/pkg/compliance/checks/hipaa_164/check308a3iib"
	"github.com/stackrox/rox/pkg/compliance/checks/hipaa_164/check308a4"
	"github.com/stackrox/rox/pkg/compliance/checks/hipaa_164/check312e1"
)

// Init registers all HIPAA 164 compliance checks.
// Called explicitly from pkg/compliance/checks/init.go instead of package init().
func Init() {
	check308a3iib.RegisterCheck308a3iib()
	check308a4.RegisterCheck308a4()
	check312e1.RegisterCheck312e1()
}
