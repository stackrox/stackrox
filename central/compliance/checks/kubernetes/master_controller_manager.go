package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		masterControllerManagerCommandLine("CIS_Kubernetes_v1_4_1:1_3_1", "terminated-pod-gc-threshold", "", "", common.Set),
		masterControllerManagerCommandLine("CIS_Kubernetes_v1_4_1:1_3_2", "profiling", "", "", common.Set),
		masterControllerManagerCommandLine("CIS_Kubernetes_v1_4_1:1_3_3", "use-service-account-credentials", "", "", common.Set),
		masterControllerManagerCommandLine("CIS_Kubernetes_v1_4_1:1_3_4", "service-account-private-key-file", "", "", common.Set),
		masterControllerManagerCommandLine("CIS_Kubernetes_v1_4_1:1_3_5", "root-ca-file", "", "", common.Set),
		masterControllerManagerCommandLine("CIS_Kubernetes_v1_4_1:1_3_6", "feature-gates", "RotateKubeletServerCertificate=true", "", common.Contains),
		masterControllerManagerCommandLine("CIS_Kubernetes_v1_4_1:1_3_7", "address", "127.0.0.1", "127.0.0.1", common.Matches),
	)
}

func masterControllerManagerCommandLine(name string, key, target, defaultVal string, evalFunc common.CommandEvaluationFunc) framework.Check {
	return genericKubernetesCommandlineCheck(name, "kube-controller-manager", key, target, defaultVal, evalFunc)
}
