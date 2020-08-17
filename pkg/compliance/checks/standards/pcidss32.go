package standards

// PCIDSS32 is the string name of this standard
const PCIDSS32 = "PCI_DSS_3_2"

// PCIDSS32CheckName is takes a check ID and returns a formatted check name
func PCIDSS32CheckName(checkName string) string {
	return CheckName(PCIDSS32, checkName)
}
