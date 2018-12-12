package hostconfiguration

import (
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type pathAudit struct {
	Name        string
	Description string
	Path        string
}

func (s *pathAudit) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        s.Name,
			Description: s.Description,
		},
	}
}

func (s *pathAudit) Run() (result storage.BenchmarkCheckResult) {
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
