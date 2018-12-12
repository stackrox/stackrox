package masterconfigurationfiles

import (
	"github.com/stackrox/rox/benchmarks/checks"
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type cniDataPermissions struct{}

func (c *cniDataPermissions) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Kubernetes v1.2.0 - 1.4.9",
			Description: "Ensure that the Container Network Interface file permissions are set to 644 or more restrictive",
		}, Dependencies: []utils.Dependency{utils.InitKubeletConfig},
	}
}

func (c *cniDataPermissions) Run() (result storage.BenchmarkCheckResult) {
	utils.Pass(&result)

	var dir string
	params, ok := utils.KubeletConfig.Get("cni-conf-dir")
	if !ok {
		dir = "/etc/cni/net.d"
	} else {
		dir = params.String()
	}
	result = utils.NewRecursivePermissionsCheck("", "", dir, 0644, true).Run()

	params, ok = utils.KubeletConfig.Get("cni-bin-dir")
	if !ok {
		dir = "/opt/cni/bin"
	} else {
		dir = params.String()
	}
	binDirRes := utils.NewRecursivePermissionsCheck("", "", dir, 0644, true).Run()

	if result.Result == storage.BenchmarkCheckStatus_PASS {
		result.Result = binDirRes.Result
	}
	utils.AddNotes(&result, binDirRes.Notes...)
	return
}

// NewCNIDataPermissions implements CIS Kubernetes v1.2.0 1.4.9
func NewCNIDataPermissions() utils.Check {
	return &cniDataPermissions{}
}

func init() {
	checks.AddToRegistry(
		NewCNIDataPermissions(),
	)
}
