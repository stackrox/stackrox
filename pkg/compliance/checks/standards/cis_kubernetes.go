package standards

// CISKubernetes is the string name of this standard
const CISKubernetes = "CIS_Kubernetes_v1_5"

// CISKubeCheckName takes a check ID and returns a formatted check name
func CISKubeCheckName(checkName string) string {
	return CheckName(CISKubernetes, checkName)
}
