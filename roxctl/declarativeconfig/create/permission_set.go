package create

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/declarativeconfig/k8sobject"
	"github.com/stackrox/rox/roxctl/declarativeconfig/lint"
	"gopkg.in/yaml.v3"
)

func permissionSetCommand(cliEnvironment environment.Environment) *cobra.Command {
	permSetCmd := &permissionSetCmd{permissionSet: &declarativeconfig.PermissionSet{}, env: cliEnvironment}

	cmd := &cobra.Command{
		Use:  permSetCmd.permissionSet.Type(),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := permSetCmd.Construct(cmd); err != nil {
				return err
			}
			if err := permSetCmd.Validate(); err != nil {
				return err
			}
			return permSetCmd.PrintYAML()
		},
		Short: "Create a declarative configuration for a permission set",
	}

	cmd.Flags().StringVar(&permSetCmd.permissionSet.Name, "name", "", "name of the permission set")
	cmd.Flags().StringVar(&permSetCmd.permissionSet.Description, "description", "",
		"description of the permission set")
	cmd.Flags().StringToStringVar(&permSetCmd.resourceWithAccess, "resource-with-access", map[string]string{},
		`list of resources with the respective access, e.g. --resource-with-access Access=READ_ACCESS,Administration=READ_WRITE_ACCESS
Note: Capitalization matters!`)

	cmd.MarkFlagsRequiredTogether("name", "resource-with-access")

	return cmd
}

type permissionSetCmd struct {
	permissionSet      *declarativeconfig.PermissionSet
	resourceWithAccess map[string]string
	env                environment.Environment

	configMap string
	secret    string
	namespace string
}

func (p *permissionSetCmd) Construct(cmd *cobra.Command) error {
	configMap, secret, namespace, err := k8sobject.ReadK8sObjectFlags(cmd)
	if err != nil {
		return errors.Wrap(err, "reading config map flag values")
	}
	p.configMap = configMap
	p.secret = secret
	p.namespace = namespace
	return nil
}

func (p *permissionSetCmd) Validate() error {
	accessMap := p.resourceWithAccess

	resourceWithAccess := make([]declarativeconfig.ResourceWithAccess, 0, len(accessMap))

	// Keep an alphabetic order within the resources.
	resources := maputil.Keys(accessMap)
	sort.Strings(resources)

	// TODO(ROX-16330): Resources are currently defined within central/role/resources, and hence cannot be reused here yet.
	// There are plans to move the resource definition to a shared place however, in which case we can reuse them here.
	var invalidAccessErrors *multierror.Error
	for _, resource := range resources {
		accessVal, ok := storage.Access_value[strings.ToUpper(accessMap[resource])]
		if !ok {
			invalidAccessErrors = multierror.Append(invalidAccessErrors, errox.InvalidArgs.
				Newf("invalid access specified for resource %s: %s. The allowed values for access are: [%s]",
					resource, accessMap[resource], strings.Join(maputil.Keys(storage.Access_value), ",")))
			continue
		}
		resourceWithAccess = append(resourceWithAccess, declarativeconfig.ResourceWithAccess{
			Resource: resource,
			Access:   declarativeconfig.Access(accessVal),
		})
	}
	p.permissionSet.Resources = resourceWithAccess
	return errors.Wrap(invalidAccessErrors.ErrorOrNil(), "validating permission set")
}

func (p *permissionSetCmd) PrintYAML() error {
	yamlOutput := &bytes.Buffer{}
	enc := yaml.NewEncoder(yamlOutput)
	if err := enc.Encode(p.permissionSet); err != nil {
		return errors.Wrap(err, "creating the YAML output")
	}
	if err := lint.Lint(yamlOutput.Bytes()); err != nil {
		return errors.Wrap(err, "linting the YAML output")
	}
	if p.configMap != "" || p.secret != "" {
		return errors.Wrap(k8sobject.WriteToK8sObject(context.Background(), p.configMap, p.secret, p.namespace,
			fmt.Sprintf("%s-%s", p.permissionSet.Type(), p.permissionSet.Name), yamlOutput.Bytes()),
			"writing the YAML output to config map")
	}

	if _, err := p.env.InputOutput().Out().Write(yamlOutput.Bytes()); err != nil {
		return errors.Wrap(err, "writing the YAML output")
	}
	return nil
}
