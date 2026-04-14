package nist800190

import (
	"github.com/stackrox/rox/central/compliance/checks/nist800-190/check411"
	"github.com/stackrox/rox/central/compliance/checks/nist800-190/check412"
	"github.com/stackrox/rox/central/compliance/checks/nist800-190/check414"
	"github.com/stackrox/rox/central/compliance/checks/nist800-190/check422"
	"github.com/stackrox/rox/central/compliance/checks/nist800-190/check431"
	"github.com/stackrox/rox/central/compliance/checks/nist800-190/check432"
	"github.com/stackrox/rox/central/compliance/checks/nist800-190/check433"
	"github.com/stackrox/rox/central/compliance/checks/nist800-190/check435"
	"github.com/stackrox/rox/central/compliance/checks/nist800-190/check442"
	"github.com/stackrox/rox/central/compliance/checks/nist800-190/check443"
	"github.com/stackrox/rox/central/compliance/checks/nist800-190/check444"
	"github.com/stackrox/rox/central/compliance/checks/nist800-190/check451"
	"github.com/stackrox/rox/central/compliance/checks/nist800-190/check455"
)

// Init registers all central NIST 800-190 compliance checks.
// Called explicitly from central/compliance/checks/all.go instead of package init().
func Init() {
	check411.Register411()
	check412.Register412()
	check414.Register414()
	check422.Register422()
	check431.Register431()
	check432.Register432()
	check433.Register433()
	check435.Register435()
	check442.Register442()
	check443.Register443()
	check444.Register444()
	check451.Register451()
	check455.Register455()
}
