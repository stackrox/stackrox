package containerruntime

import (
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/checks/utils"
)

type acquiringPrivilegesBenchmark struct{}

func (c *acquiringPrivilegesBenchmark) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        "CIS Docker v1.1.0 - 5.25",
			Description: "Ensure the container is restricted from acquiring additional privileges",
		}, Dependencies: []utils.Dependency{utils.InitContainers},
	}
}

func (c *acquiringPrivilegesBenchmark) Run() (result v1.CheckResult) {
	utils.Pass(&result)
LOOP:
	for _, container := range utils.ContainersRunning {
		for _, opt := range container.HostConfig.SecurityOpt {
			if strings.Contains(opt, "no-new-privileges") {
				continue LOOP
			}
		}
		utils.Warn(&result)
		utils.AddNotef(&result, "Container '%v' (%v) does not set no-new-privileges in security opts", container.ID, container.Name)
	}
	return
}

// NewAcquiringPrivilegesBenchmark implements CIS-5.25
func NewAcquiringPrivilegesBenchmark() utils.Check {
	return &acquiringPrivilegesBenchmark{}
}
