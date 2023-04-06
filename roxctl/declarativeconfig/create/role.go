package create

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/declarativeconfig/lint"
	"gopkg.in/yaml.v3"
)

func roleCommand(cliEnvironment environment.Environment) *cobra.Command {
	roleCmd := &roleCmd{role: &declarativeconfig.Role{}, env: cliEnvironment}

	cmd := &cobra.Command{
		Use:  roleCmd.role.Type(),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return roleCmd.PrintYAML()
		},
		Short: "Create a declarative configuration for a role",
	}

	cmd.Flags().StringVar(&roleCmd.role.Name, "name", "", "name of the role")
	cmd.Flags().StringVar(&roleCmd.role.Description, "description", "", "description of the role")
	cmd.Flags().StringVar(&roleCmd.role.AccessScope, "access-scope", "",
		"name of the referenced access scope")
	cmd.Flags().StringVar(&roleCmd.role.PermissionSet, "permission-set", "",
		"name of the referenced permission set")

	// No additional validation is required for roles, since a role is valid when name, permission set, access
	// scope are set, which is covered by requiring the flag.
	utils.Must(cmd.MarkFlagRequired("name"))
	utils.Must(cmd.MarkFlagRequired("access-scope"))
	utils.Must(cmd.MarkFlagRequired("permission-set"))
	return cmd
}

type roleCmd struct {
	role *declarativeconfig.Role
	env  environment.Environment
}

func (r *roleCmd) PrintYAML() error {
	yamlOutput := &bytes.Buffer{}
	enc := yaml.NewEncoder(yamlOutput)
	if err := enc.Encode(r.role); err != nil {
		return errors.Wrap(err, "creating the YAML output")
	}
	if err := lint.Lint(yamlOutput.Bytes()); err != nil {
		return errors.Wrap(err, "linting the YAML output")
	}
	_, err := r.env.InputOutput().Out().Write(yamlOutput.Bytes())
	return errors.Wrap(err, "writing the YAML output")
}
