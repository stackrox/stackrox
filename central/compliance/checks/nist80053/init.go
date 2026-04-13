package nist80053

import (
	checkac14 "github.com/stackrox/rox/central/compliance/checks/nist80053/check_ac_14"
	checkca9 "github.com/stackrox/rox/central/compliance/checks/nist80053/check_ca_9"
	checkcm11 "github.com/stackrox/rox/central/compliance/checks/nist80053/check_cm_11"
	checkcm2 "github.com/stackrox/rox/central/compliance/checks/nist80053/check_cm_2"
	checkcm3 "github.com/stackrox/rox/central/compliance/checks/nist80053/check_cm_3"
	checkcm5 "github.com/stackrox/rox/central/compliance/checks/nist80053/check_cm_5"
	checkcm6 "github.com/stackrox/rox/central/compliance/checks/nist80053/check_cm_6"
	checkcm7 "github.com/stackrox/rox/central/compliance/checks/nist80053/check_cm_7"
	checkcm8 "github.com/stackrox/rox/central/compliance/checks/nist80053/check_cm_8"
	checkir45 "github.com/stackrox/rox/central/compliance/checks/nist80053/check_ir_4_5"
	checkir5 "github.com/stackrox/rox/central/compliance/checks/nist80053/check_ir_5"
	checkir61 "github.com/stackrox/rox/central/compliance/checks/nist80053/check_ir_6_1"
	checkra3 "github.com/stackrox/rox/central/compliance/checks/nist80053/check_ra_3"
	checkra5 "github.com/stackrox/rox/central/compliance/checks/nist80053/check_ra_5"
	checksa10 "github.com/stackrox/rox/central/compliance/checks/nist80053/check_sa_10"
	checksc6 "github.com/stackrox/rox/central/compliance/checks/nist80053/check_sc_6"
	checksc7 "github.com/stackrox/rox/central/compliance/checks/nist80053/check_sc_7"
	checksi22 "github.com/stackrox/rox/central/compliance/checks/nist80053/check_si_2_2"
	checksi38 "github.com/stackrox/rox/central/compliance/checks/nist80053/check_si_3_8"
	checksi4 "github.com/stackrox/rox/central/compliance/checks/nist80053/check_si_4"
)

// Init registers all central NIST 800-53 compliance checks.
// Called explicitly from central/compliance/checks/all.go instead of package init().
func Init() {
	checkac14.RegisterAC14()
	checkca9.RegisterCA9()
	checkcm11.RegisterCM11()
	checkcm2.RegisterCM2()
	checkcm3.RegisterCM3()
	checkcm5.RegisterCM5()
	checkcm6.RegisterCM6()
	checkcm7.RegisterCM7()
	checkcm8.RegisterCM8()
	checkir45.RegisterIR45()
	checkir5.RegisterIR5()
	checkir61.RegisterIR61()
	checkra3.RegisterRA3()
	checkra5.RegisterRA5()
	checksa10.RegisterSA10()
	checksc6.RegisterSC6()
	checksc7.RegisterSC7()
	checksi22.RegisterSI22()
	checksi38.RegisterSI38()
	checksi4.RegisterSI4()
}
