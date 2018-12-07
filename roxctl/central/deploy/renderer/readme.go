package renderer

import (
	"text/template"

	"github.com/stackrox/rox/pkg/templates"
)

const readmeTemplateText = `
{{- .DeployerInstructions }}

For administrator login, select the "Login with username/password" option on
the login page, and log in with username "admin" and the password found in the
"password" file located in the same directory as this README.
`

var (
	readmeTemplate = template.Must(template.New("readme").Parse(readmeTemplateText))
)

// generateReadme generates a README file.
func generateReadme(c *Config) (string, error) {
	return templates.ExecuteToString(readmeTemplate, map[string]interface{}{
		"DeployerInstructions": Deployers[c.ClusterType].Instructions(*c),
		"Config":               c,
	})
}
