package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		masterSchedulerCommandLine("CIS_Kubernetes_v1_2_0:1_2_1", "profiling", "false", "true", common.Matches),
	)
}

func masterSchedulerCommandLine(name string, key, target, defaultVal string, evalFunc common.CommandEvaluationFunc) framework.Check {
	return genericKubernetesCommandlineCheck(name, "kube-scheduler", key, target, defaultVal, evalFunc)
}
