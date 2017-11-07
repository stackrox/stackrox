package dockerdaemonconfiguration

import (
	"context"
	"strings"

	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type insecureRegistriesBenchmark struct{}

func (c *insecureRegistriesBenchmark) Definition() common.Definition {
	return common.Definition{
		Name:         "CIS 2.4",
		Description:  "Ensure insecure registries are not used",
		Dependencies: []common.Dependency{common.InitDockerClient},
	}
}

func (c *insecureRegistriesBenchmark) Run() (result common.TestResult) {
	result.Pass()
	info, err := common.DockerClient.Info(context.Background())
	if err != nil {
		result.Warn()
		result.AddNotes(err.Error())
		return
	}
	for _, registry := range info.RegistryConfig.InsecureRegistryCIDRs {
		if strings.HasPrefix(registry.String(), "127.") { // Localhost prefix can be ignored
			continue
		}
		result.Warn()
		result.AddNotef("Insecure registry with CIDR %v is configured", registry)
	}
	return
}

// NewInsecureRegistriesBenchmark implements CIS-2.4
func NewInsecureRegistriesBenchmark() common.Benchmark {
	return &insecureRegistriesBenchmark{}
}
