package masterconfigurationfiles

import (
	"github.com/stackrox/rox/benchmarks/checks"
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type cniDataOwnership struct{}

func (c *cniDataOwnership) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Kubernetes v1.2.0 - 1.4.10",
			Description: "Ensure that the Container Network Interface file ownership is set to root:root",
		}, Dependencies: []utils.Dependency{utils.InitKubeletConfig},
	}
}

func (c *cniDataOwnership) Run() (result v1.CheckResult) {
	var dir string
	params, ok := utils.KubeletConfig.Get("cni-conf-dir")
	if !ok {
		dir = "/etc/cni/net.d"
	} else {
		dir = params.String()
	}
	result = utils.NewRecursiveOwnershipCheck("", "", dir, "root", "root").Run()

	params, ok = utils.KubeletConfig.Get("cni-bin-dir")
	if !ok {
		dir = "/opt/cni/bin"
	} else {
		dir = params.String()
	}
	binDirRes := utils.NewRecursiveOwnershipCheck("", "", dir, "root", "root").Run()

	if result.Result == v1.CheckStatus_PASS {
		result.Result = binDirRes.Result
	}
	utils.AddNotes(&result, binDirRes.Notes...)
	return
}

// NewCNIDataOwnership implements CIS Kubernetes v1.2.0 1.4.10
func NewCNIDataOwnership() utils.Check {
	return &cniDataOwnership{}
}

func init() {
	checks.AddToRegistry(
		NewCNIDataOwnership(),
	)
}
