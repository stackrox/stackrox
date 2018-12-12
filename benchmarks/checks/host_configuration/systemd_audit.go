package hostconfiguration

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type systemdAudit struct {
	Name        string
	Description string
	Service     string
}

func (s *systemdAudit) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        s.Name,
			Description: s.Description,
		},
	}
}

func (s *systemdAudit) Run() (result storage.BenchmarkCheckResult) {
	path, err := utils.GetSystemdFile(s.Service)
	if err != nil {
		utils.Note(&result)
		utils.AddNotef(&result, "Test may not be applicable. Systemd file could not be found for service %v", s.Service)
		return
	}
	result = utils.CheckAudit(path)
	return
}

func newSystemdAudit(name, description, service string) utils.Check {
	return &systemdAudit{
		Name:        name,
		Description: description,
		Service:     service,
	}
}
