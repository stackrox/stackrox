package hipaa164

import (
	"github.com/stackrox/rox/central/compliance/checks/hipaa_164/check306e"
	"github.com/stackrox/rox/central/compliance/checks/hipaa_164/check308a1i"
	"github.com/stackrox/rox/central/compliance/checks/hipaa_164/check308a1iia"
	"github.com/stackrox/rox/central/compliance/checks/hipaa_164/check308a1iib"
	"github.com/stackrox/rox/central/compliance/checks/hipaa_164/check308a3iia"
	"github.com/stackrox/rox/central/compliance/checks/hipaa_164/check308a3iib"
	"github.com/stackrox/rox/central/compliance/checks/hipaa_164/check308a4"
	"github.com/stackrox/rox/central/compliance/checks/hipaa_164/check308a4iib"
	"github.com/stackrox/rox/central/compliance/checks/hipaa_164/check308a5iib"
	"github.com/stackrox/rox/central/compliance/checks/hipaa_164/check308a6ii"
	"github.com/stackrox/rox/central/compliance/checks/hipaa_164/check308a7iie"
	"github.com/stackrox/rox/central/compliance/checks/hipaa_164/check310a1"
	"github.com/stackrox/rox/central/compliance/checks/hipaa_164/check310a1a"
	"github.com/stackrox/rox/central/compliance/checks/hipaa_164/check310d"
	"github.com/stackrox/rox/central/compliance/checks/hipaa_164/check312c"
	"github.com/stackrox/rox/central/compliance/checks/hipaa_164/check312e"
	"github.com/stackrox/rox/central/compliance/checks/hipaa_164/check312e1"
	"github.com/stackrox/rox/central/compliance/checks/hipaa_164/check314a2ic"
	check316b2iii "github.com/stackrox/rox/central/compliance/checks/hipaa_164/check316b2iii"
)

// Init registers all central HIPAA 164 compliance checks.
// Called explicitly from central/compliance/checks/all.go instead of package init().
func Init() {
	check306e.Register306e()
	check308a1i.Register308a1i()
	check308a1iia.Register308a1iia()
	check308a1iib.Register308a1iib()
	check308a3iia.Register308a3iia()
	check308a3iib.Register308a3iib()
	check308a4.Register308a4()
	check308a4iib.Register308a4iib()
	check308a5iib.Register308a5iib()
	check308a6ii.Register308a6ii()
	check308a7iie.Register308a7iie()
	check310a1.Register310a1()
	check310a1a.Register310a1a()
	check310d.Register310d()
	check312c.Register312c()
	check312e.Register312e()
	check312e1.Register312e1()
	check314a2ic.Register314a2ic()
	check316b2iii.Register316b2iii()
}
