package pcidss32

import (
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check112"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check1121"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check114"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check12"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check121"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check132"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check134"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check135"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check21"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check22"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check225"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check23"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check24"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check362"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check61"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check62"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check656"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check71"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check712"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check713"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check722"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check723"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check811"
	"github.com/stackrox/rox/central/compliance/checks/pcidss32/check85"
)

// Init registers all central PCI DSS 3.2 compliance checks.
// Called explicitly from central/compliance/checks/all.go instead of package init().
func Init() {
	check112.Register112()
	check1121.Register1121()
	check114.Register114()
	check12.Register12()
	check121.Register121()
	check132.Register132()
	check134.Register134()
	check135.Register135()
	check21.Register21()
	check22.Register22()
	check225.Register225()
	check23.Register23()
	check24.Register24()
	check362.Register362()
	check61.Register61()
	check62.Register62()
	check656.Register656()
	check71.Register71()
	check712.Register712()
	check713.Register713()
	check722.Register722()
	check723.Register723()
	check811.Register811()
	check85.Register85()
}
