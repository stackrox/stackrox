package pcidss32

import (
	"github.com/stackrox/rox/pkg/compliance/checks/pcidss32/check362"
	"github.com/stackrox/rox/pkg/compliance/checks/pcidss32/check71"
	"github.com/stackrox/rox/pkg/compliance/checks/pcidss32/check712"
	"github.com/stackrox/rox/pkg/compliance/checks/pcidss32/check713"
	"github.com/stackrox/rox/pkg/compliance/checks/pcidss32/check722"
	"github.com/stackrox/rox/pkg/compliance/checks/pcidss32/check723"
	"github.com/stackrox/rox/pkg/compliance/checks/pcidss32/check811"
	"github.com/stackrox/rox/pkg/compliance/checks/pcidss32/check85"
)

// Init registers all PCI DSS 3.2 compliance checks.
// Called explicitly from pkg/compliance/checks/init.go instead of package init().
func Init() {
	check362.RegisterCheck362()
	check71.RegisterCheck71()
	check712.RegisterCheck712()
	check713.RegisterCheck713()
	check722.RegisterCheck722()
	check723.RegisterCheck723()
	check811.RegisterCheck811()
	check85.RegisterCheck85()
}
