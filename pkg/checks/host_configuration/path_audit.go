package hostconfiguration

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
)

type pathAudit struct {
	Name        string
	Description string
	Path        string
}

func (s *pathAudit) Definition() utils.Definition {
	return utils.Definition{
		CheckDefinition: v1.CheckDefinition{
			Name:        s.Name,
			Description: s.Description,
		},
	}
}

func (s *pathAudit) Run() (result v1.CheckResult) {
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
