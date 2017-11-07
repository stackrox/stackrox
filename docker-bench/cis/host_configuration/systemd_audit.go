package hostconfiguration

import (
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

type systemdAudit struct {
	Name        string
	Description string
	Service     string
}

func (s *systemdAudit) Definition() common.Definition {
	return common.Definition{
		Name:        s.Name,
		Description: s.Description,
	}
}

func (s *systemdAudit) Run() (result common.TestResult) {
	path := common.GetSystemdFile(s.Service)
	result = common.CheckAudit(path)
	return
}

func newSystemdAudit(name, description, service string) common.Benchmark {
	return &systemdAudit{
		Name:        name,
		Description: description,
		Service:     service,
	}
}
