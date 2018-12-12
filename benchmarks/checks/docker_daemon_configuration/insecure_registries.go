package dockerdaemonconfiguration

import (
	"strings"

	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type insecureRegistriesBenchmark struct{}

func (c *insecureRegistriesBenchmark) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 2.4",
			Description: "Ensure insecure registries are not used",
		}, Dependencies: []utils.Dependency{utils.InitInfo},
	}
}

func (c *insecureRegistriesBenchmark) Run() (result storage.BenchmarkCheckResult) {
	utils.Pass(&result)
	for _, registry := range utils.DockerInfo.RegistryConfig.InsecureRegistryCIDRs {
		if strings.HasPrefix(registry.String(), "127.") { // Localhost prefix can be ignored
			continue
		}
		utils.Warn(&result)
		utils.AddNotef(&result, "Insecure registry with CIDR '%v' is configured", registry)
	}
	return
}

// NewInsecureRegistriesBenchmark implements CIS-2.4
func NewInsecureRegistriesBenchmark() utils.Check {
	return &insecureRegistriesBenchmark{}
}
