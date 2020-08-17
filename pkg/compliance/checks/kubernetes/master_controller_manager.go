package kubernetes

import (
	"github.com/stackrox/rox/pkg/compliance/checks/common"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
)

func init() {
	standards.RegisterChecksForStandard(standards.CISKubernetes, map[string]*standards.CheckAndMetadata{
		standards.CISKubeCheckName("1_3_1"): masterControllerManagerCommandLine("terminated-pod-gc-threshold", "", "12500", common.Set),
		standards.CISKubeCheckName("1_3_2"): masterControllerManagerCommandLine("profiling", "false", "true", common.Matches),
		standards.CISKubeCheckName("1_3_3"): masterControllerManagerCommandLine("use-service-account-credentials", "true", "true", common.Matches),
		standards.CISKubeCheckName("1_3_4"): masterControllerManagerCommandLine("service-account-private-key-file", "", "", common.Set),
		standards.CISKubeCheckName("1_3_5"): masterControllerManagerCommandLine("root-ca-file", "", "/etc/kubernetes/pki/ca.crt", common.Set),
		standards.CISKubeCheckName("1_3_6"): masterControllerManagerCommandLine("feature-gates", "RotateKubeletServerCertificate=true", "", common.Contains),
		standards.CISKubeCheckName("1_3_7"): masterControllerManagerCommandLine("bind-address", "127.0.0.1", "127.0.0.1", common.Matches),
	})
}

func masterControllerManagerCommandLine(key, target, defaultVal string, evalFunc common.CommandEvaluationFunc) *standards.CheckAndMetadata {
	return common.MasterNodeKubernetesCommandlineCheck("kube-controller-manager", key, target, defaultVal, evalFunc)
}
