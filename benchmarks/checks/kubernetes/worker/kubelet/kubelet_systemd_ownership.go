package kubelet

import (
	"github.com/stackrox/rox/benchmarks/checks"
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type kubeletSystemdOwnership struct{}

func (c *kubeletSystemdOwnership) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Kubernetes v1.2.0 - 2.2.4",
			Description: "Ensure that the kubelet service file ownership is set to root:root",
		},
	}
}

func (c *kubeletSystemdOwnership) Run() (result v1.CheckResult) {
	result = utils.NewSystemdOwnershipCheck("", "", "kubelet.service", "root", "root").Run()
	return
}

// NewKubeletSystemdOwnership implements CIS Kubernetes v1.2.0 2.2.4
func NewKubeletSystemdOwnership() utils.Check {
	return &kubeletSystemdOwnership{}
}

func init() {
	checks.AddToRegistry(
		NewKubeletSystemdOwnership(),
	)
}
