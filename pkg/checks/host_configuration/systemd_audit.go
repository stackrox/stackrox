package hostconfiguration

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
)

type systemdAudit struct {
	Name        string
	Description string
	Service     string
}

func (s *systemdAudit) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        s.Name,
			Description: s.Description,
		},
	}
}

func (s *systemdAudit) Run() (result v1.CheckResult) {
	path := utils.GetSystemdFile(s.Service)
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
