package hostconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/utils"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type pathAudit struct {
	Name        string
	Description string
	Path        string
}

func (s *pathAudit) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkDefinition: v1.BenchmarkDefinition{
			Name:        s.Name,
			Description: s.Description,
		},
	}
}

func (s *pathAudit) Run() (result v1.BenchmarkTestResult) {
	result = utils.CheckAudit(s.Path)
	return
}

func newPathAudit(name, description, path string) utils.Benchmark {
	return &pathAudit{
		Name:        name,
		Description: description,
		Path:        path,
	}
}
