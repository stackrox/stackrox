package kubernetes

import (
	"github.com/stackrox/rox/pkg/compliance/checks/common"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
)

func init() {
	standards.RegisterChecksForStandard(standards.CISKubernetes, map[string]*standards.CheckAndMetadata{
		standards.CISKubeCheckName("1_4_1"): masterSchedulerCommandLine("profiling", "false", "true", common.Matches),
		standards.CISKubeCheckName("1_4_2"): masterSchedulerCommandLine("bind-address", "127.0.0.1", "127.0.0.1", common.Matches),
	})
}

func masterSchedulerCommandLine(key, target, defaultVal string, evalFunc common.CommandEvaluationFunc) *standards.CheckAndMetadata {
	return common.MasterNodeKubernetesCommandlineCheck("kube-scheduler", key, target, defaultVal, evalFunc)
}
