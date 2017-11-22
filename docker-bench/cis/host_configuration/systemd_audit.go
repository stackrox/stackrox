package hostconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type systemdAudit struct {
	Name        string
	Description string
	Service     string
}

func (s *systemdAudit) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        s.Name,
			Description: s.Description,
		},
	}
}

func (s *systemdAudit) Run() (result v1.BenchmarkTestResult) {
	path := utils.GetSystemdFile(s.Service)
	result = utils.CheckAudit(path)
	return
}

func newSystemdAudit(name, description, service string) utils.Benchmark {
	return &systemdAudit{
		Name:        name,
		Description: description,
		Service:     service,
	}
}
