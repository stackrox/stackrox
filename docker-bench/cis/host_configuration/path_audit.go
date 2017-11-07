package hostconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type pathAudit struct {
	Name        string
	Description string
	Path        string
}

func (s *pathAudit) Definition() common.Definition {
	return common.Definition{
		Name:        s.Name,
		Description: s.Description,
	}
}

func (s *pathAudit) Run() (result common.TestResult) {
	result = common.CheckAudit(s.Path)
	return
}

func newPathAudit(name, description, path string) common.Benchmark {
	return &pathAudit{
		Name:        name,
		Description: description,
		Path:        path,
	}
}
