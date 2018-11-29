package hostconfiguration

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
)

type pathAudit struct {
	Name        string
	Description string
	Path        string
}

func (s *pathAudit) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        s.Name,
			Description: s.Description,
		},
	}
}

func (s *pathAudit) Run() (result v1.BenchmarkCheckResult) {
	result = utils.CheckAudit(s.Path)
	return
}

func newPathAudit(name, description, path string) utils.Check {
	return &pathAudit{
		Name:        name,
		Description: description,
		Path:        path,
	}
}
