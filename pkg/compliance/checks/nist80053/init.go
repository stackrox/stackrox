package nist80053

import (
	checkac14 "github.com/stackrox/rox/pkg/compliance/checks/nist80053/check_ac_14"
	checkac24 "github.com/stackrox/rox/pkg/compliance/checks/nist80053/check_ac_24"
	checkac37 "github.com/stackrox/rox/pkg/compliance/checks/nist80053/check_ac_3_7"
	checkcm5 "github.com/stackrox/rox/pkg/compliance/checks/nist80053/check_cm_5"
)

// Init registers all NIST 800-53 compliance checks.
// Called explicitly from pkg/compliance/checks/init.go instead of package init().
func Init() {
	checkac14.RegisterCheckAC14()
	checkac24.RegisterCheckAC24()
	checkac37.RegisterCheckAC37()
	checkcm5.RegisterCheckCM5()
}
