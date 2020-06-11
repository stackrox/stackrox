package standards

// NIST is the string name of this standard
const NIST = "NIST_800_190"

// NISTCheckName is takes a check ID and returns a formatted check name
func NISTCheckName(checkName string) string {
	return CheckName(NIST, checkName)
}
