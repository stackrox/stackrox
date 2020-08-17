package standards

// NIST800190 is the string name of this standard
const NIST800190 = "NIST_800_190"

// NIST800190CheckName is takes a check ID and returns a formatted check name
func NIST800190CheckName(checkName string) string {
	return CheckName(NIST800190, checkName)
}
