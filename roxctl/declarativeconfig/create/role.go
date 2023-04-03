package create

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/declarativeconfig/transform"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"gopkg.in/yaml.v3"
)

func roleCommand(cliEnvironment environment.Environment) *cobra.Command {
	roleCmd := &roleCmd{role: &declarativeconfig.Role{}, env: cliEnvironment}

	cmd := &cobra.Command{
		Use:  roleCmd.role.Type(),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := roleCmd.Validate(); err != nil {
				return err
			}
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

func (r *roleCmd) Validate() error {
	t := transform.New()
	_, err := t.Transform(r.role)
	return errors.Wrap(err, "validate role")
}

func (r *roleCmd) PrintYAML() error {
	enc := yaml.NewEncoder(r.env.InputOutput().Out())
	return errors.Wrap(enc.Encode(r.role), "creating the YAML output")
}
