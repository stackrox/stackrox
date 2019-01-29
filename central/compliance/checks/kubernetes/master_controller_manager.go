package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		masterControllerManagerCommandLine("CIS_Kubernetes_v1_2_0:1_3_1", "terminated-pod-gc-threshold", "", "", common.Set),
		masterControllerManagerCommandLine("CIS_Kubernetes_v1_2_0:1_3_2", "profiling", "", "", common.Set),
		masterControllerManagerCommandLine("CIS_Kubernetes_v1_2_0:1_3_3", "use-service-account-credentials", "", "", common.Set),
		masterControllerManagerCommandLine("CIS_Kubernetes_v1_2_0:1_3_4", "service-account-private-key-file", "", "", common.Set),
		masterControllerManagerCommandLine("CIS_Kubernetes_v1_2_0:1_3_5", "root-ca-file", "", "", common.Set),

		common.PerNodeNoteCheck("CIS_Kubernetes_v1_2_0:1_3_6", "Apply Security Context to Your Pods and Containers"),
		masterControllerManagerCommandLine("CIS_Kubernetes_v1_2_0:1_3_7", "feature-gates", "RotateKubeletServerCertificate=true", "", common.Contains),
	)
}

func masterControllerManagerCommandLine(name string, key, target, defaultVal string, evalFunc common.CommandEvaluationFunc) framework.Check {
	return genericKubernetesCommandlineCheck(name, "kube-controller-manager", key, target, defaultVal, evalFunc)
}
