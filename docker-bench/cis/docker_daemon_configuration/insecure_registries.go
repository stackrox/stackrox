package dockerdaemonconfiguration

import (
	"context"
	"strings"

	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type insecureRegistriesBenchmark struct{}

func (c *insecureRegistriesBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS 2.4",
			Description: "Ensure insecure registries are not used",
		}, Dependencies: []utils.Dependency{utils.InitDockerClient},
	}
}

func (c *insecureRegistriesBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
	info, err := utils.DockerClient.Info(context.Background())
	if err != nil {
		utils.Warn(&result)
		utils.AddNotes(&result, err.Error())
		return
	}
	for _, registry := range info.RegistryConfig.InsecureRegistryCIDRs {
		if strings.HasPrefix(registry.String(), "127.") { // Localhost prefix can be ignored
			continue
		}
		utils.Warn(&result)
		utils.AddNotef(&result, "Insecure registry with CIDR %v is configured", registry)
	}
	return
}

// NewInsecureRegistriesBenchmark implements CIS-2.4
func NewInsecureRegistriesBenchmark() utils.Check {
	return &insecureRegistriesBenchmark{}
}
