package masterconfigurationfiles

import (
	"github.com/stackrox/rox/benchmarks/checks"
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type etcdDataPermissions struct{}

func (c *etcdDataPermissions) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Kubernetes v1.2.0 - 1.4.11",
			Description: "Ensure that the etcd data directory permissions are set to 700 or more restrictive",
		}, Dependencies: []utils.Dependency{utils.InitEtcdConfig},
	}
}

func (c *etcdDataPermissions) Run() (result v1.BenchmarkCheckResult) {
	utils.Pass(&result)
	params, ok := utils.EtcdConfig.Get("data-dir")
	if !ok {
		utils.Note(&result)
		utils.AddNotes(&result, "Cannot check etcd data permission because etcd command line does not define 'data-dir' parameter")
		return
	}

	result = utils.NewPermissionsCheck("", "", params.String(), 0700, true).Run()
	return
}

// NewEtcdDataPermissions implements CIS Kubernetes v1.2.0 1.4.11
func NewEtcdDataPermissions() utils.Check {
	return &etcdDataPermissions{}
}

func init() {
	checks.AddToRegistry(
		NewEtcdDataPermissions(),
	)
}
