package masterconfigurationfiles

import (
	"github.com/stackrox/rox/benchmarks/checks"
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type etcdDataOwnership struct{}

func (c *etcdDataOwnership) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Kubernetes v1.2.0 - 1.4.12",
			Description: "Ensure that the etcd data directory ownership is set to etcd:etcd",
		}, Dependencies: []utils.Dependency{utils.InitEtcdConfig},
	}
}

func (c *etcdDataOwnership) Run() (result v1.BenchmarkCheckResult) {
	utils.Pass(&result)
	var dir string
	params, ok := utils.EtcdConfig.Get("data-dir")
	if ok {
		dir = params.String()
	} else {
		dir = "/var/lib/etcddisk"
	}
	result = utils.NewRecursiveOwnershipCheck("", "", dir, "etcd", "etcd").Run()
	return
}

// NewEtcdDataOwnership implements CIS Kubernetes v1.2.0 1.4.12
func NewEtcdDataOwnership() utils.Check {
	return &etcdDataOwnership{}
}

func init() {
	checks.AddToRegistry(
		NewEtcdDataOwnership(),
	)
}
