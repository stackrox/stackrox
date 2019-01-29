package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		federatedControllerManagerCommandLine("CIS_Kubernetes_v1_2_0:3_2_1", "profiling", "true", "false", common.Matches),
	)
}

func federatedControllerManagerCommandLine(name string, key, target, defaultVal string, evalFunc common.CommandEvaluationFunc) framework.Check {
	return genericKubernetesCommandlineCheck(name, "federation-controller-manager", key, target, defaultVal, evalFunc)
}
