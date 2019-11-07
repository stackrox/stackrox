package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		masterSchedulerCommandLine("CIS_Kubernetes_v1_5:1_4_1", "profiling", "false", "true", common.Matches),
		masterSchedulerCommandLine("CIS_Kubernetes_v1_5:1_4_2", "bind-address", "127.0.0.1", "127.0.0.1", common.Matches),
	)
}

func masterSchedulerCommandLine(name string, key, target, defaultVal string, evalFunc common.CommandEvaluationFunc) framework.Check {
	return genericKubernetesCommandlineCheck(name, "kube-scheduler", key, target, defaultVal, evalFunc)
}
