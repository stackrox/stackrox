package standards

// CISDocker is the string name of this standard
const CISDocker = "CIS_Docker_v1_2_0"

// CISDockerCheckName takes a check ID and returns a formetted check name
func CISDockerCheckName(checkName string) string {
	return CheckName(CISDocker, checkName)
}
