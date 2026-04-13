package nist800190

import (
	"github.com/stackrox/rox/pkg/compliance/checks/nist800-190/check421"
	"github.com/stackrox/rox/pkg/compliance/checks/nist800-190/check431"
	"github.com/stackrox/rox/pkg/compliance/checks/nist800-190/check432"
)

// Init registers all NIST 800-190 compliance checks.
// Called explicitly from pkg/compliance/checks/init.go instead of package init().
func Init() {
	check421.RegisterCheck421()
	check431.RegisterCheck431()
	check432.RegisterCheck432()
}
