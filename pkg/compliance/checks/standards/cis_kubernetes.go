package standards

// CISKubernetes is the string name of this check
const CISKubernetes = "CIS_Kubernetes_v1_5"

func init() {
	//RegisterChecksForStandard(CISKubernetes, map[string]*standards.CheckAndInterpretation{
	//	CISKubeCheckName("1_2_13"): kubernetes.SecurityContextDenyChecker,
	//	CISKubeCheckName("1_2_34"): kubernetes.EncryptionProvider,
	//
	//	CISKubeCheckName("4_1_3"): common.CommandLineFilePermissions("kubelet", "kubeconfig", 0644),
	//	CISKubeCheckName("4_1_4"):  common.CommandLineFileOwnership("kubelet", "kubeconfig", "root", "root"),
	//
	//	CISKubeCheckName("4_1_7"): common.CommandLineFilePermissions("kubelet", "client-ca-file", 0644),
	//	CISKubeCheckName("4_1_8"):  common.CommandLineFileOwnership("kubelet", "client-ca-file", "root", "root"),
	//
	//	CISKubeCheckName("4_1_9"): common.CommandLineFilePermissions("kubelet", "config", 0644),
	//	CISKubeCheckName("4_1_10"): common.CommandLineFileOwnership("kubelet", "config", "root", "root"),
	//})
}

// CISKubeCheckName takes a check ID and returns a formatted check name
func CISKubeCheckName(checkName string) string {
	return CheckName(CISKubernetes, checkName)
}
