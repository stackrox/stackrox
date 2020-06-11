package standards

// CISDocker is the string name of this standard
const CISDocker = "CIS_Docker_v1_2_0"

func init() {
	//RegisterChecksForStandard(CISKubernetes, map[string]*standards.CheckAndInterpretation{
	//	CISDockerCheckName("3_9"):  common.CommandLineFileOwnership("dockerd", "tlscacert", "root", "root"),
	//	CISDockerCheckName("3_10"): common.CommandLineFilePermissions("dockerd", "tlscacert", 0444),
	//
	//	CISDockerCheckName("3_11"): common.CommandLineFileOwnership("dockerd", "tlscert", "root", "root"),
	//	CISDockerCheckName("3_12"): common.CommandLineFilePermissions("dockerd", "tlscert", 0444),
	//
	//	CISDockerCheckName("3_13"): common.CommandLineFileOwnership("dockerd", "tlskey", "root", "root"),
	//	CISDockerCheckName("3_14"): common.CommandLineFilePermissions("dockerd", "tlskey", 0400),
	//})
}

// CISDockerCheckName takes a check ID and returns a formetted check name
func CISDockerCheckName(checkName string) string {
	return CheckName(CISDocker, checkName)
}
