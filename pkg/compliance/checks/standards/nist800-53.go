package standards

// NIST80053 is the string name of this standard
var NIST80053 = "NIST_SP_800_53_Rev_4"

// NIST80053CheckName is takes a check ID and returns a formatted check name
func NIST80053CheckName(checkName string) string {
	return CheckName(NIST80053, checkName)
}
