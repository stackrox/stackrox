package kubelet

import (
	"github.com/stackrox/rox/benchmarks/checks"
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type kubeconfigFileOwnership struct{}

func (c *kubeconfigFileOwnership) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Kubernetes v1.2.0 - 2.2.6",
			Description: "Ensure that the proxy kubeconfig file ownership is set to root:root",
		}, Dependencies: []utils.Dependency{utils.InitKubeProxyConfig},
	}
}

func (c *kubeconfigFileOwnership) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	params, ok := utils.KubeletConfig.Get("kubeconfig")
	if !ok {
		utils.Warn(&result)
		utils.AddNotes(&result, "Cannot check kubeconfig file ownership because kube-proxy command line does not define 'kubeconfig' parameter")
		return
	}

	result = utils.NewOwnershipCheck("", "", params.String(), "root", "root").Run()
	return
}

// NewKubeconfigFileOwnership implements CIS Kubernetes v1.2.0 2.2.6
func NewKubeconfigFileOwnership() utils.Check {
	return &kubeconfigFileOwnership{}
}

func init() {
	checks.AddToRegistry(
		NewKubeconfigFileOwnership(),
	)
}
