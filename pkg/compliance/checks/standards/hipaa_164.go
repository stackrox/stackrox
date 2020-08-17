package standards

// Hipaa164 is the string name of this standard
const Hipaa164 = "HIPAA_164"

// HIPAA164CheckName takes a check ID and returns a formatted check name
func HIPAA164CheckName(checkName string) string {
	return CheckName(Hipaa164, checkName)
}
