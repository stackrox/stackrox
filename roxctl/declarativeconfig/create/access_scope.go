package create

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/accessscope"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/declarativeconfig/transform"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/declarativeconfig/k8sobject"
	"github.com/stackrox/rox/roxctl/declarativeconfig/lint"
	"gopkg.in/yaml.v3"
)

var (
	_ pflag.Value = (*includedObjectsFlag)(nil)
	_ pflag.Value = (*requirementFlag)(nil)
)

const labelSelectorUsage = `The flag consists of three key value pairs with the keys:
key, value, operator. The key value pairs are expected to be separated with ;.
Each tuple represents a requirement, which will be used to construct the label selector.
You may specify this flag multiple times, to create a conjunction of requirements which should apply for a label selector
to match.

Example of a label selector requiring values: --%s "key=kubernetes.io/hostname;operator=IN;values=nodeA,nodeB"
Example of a label selector not requiring values: --%s "key=custom-label;operator=EXISTS"

NOTE: The created access scope will only contain a single label selector, where each specified requirement
will be in conjunction. If you desire to create multiple label selectors, you have to adjust the YAML output manually.
`

func accessScopeCommand(cliEnvironment environment.Environment) *cobra.Command {
	accessScopeCmd := accessScopeCmd{accessScope: &declarativeconfig.AccessScope{}, env: cliEnvironment}

	cmd := &cobra.Command{
		Use:   accessScopeCmd.accessScope.Type(),
		Short: "Create a declarative configuration for an access scope",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := accessScopeCmd.Construct(cmd); err != nil {
				return err
			}
			if err := accessScopeCmd.Validate(); err != nil {
				return err
			}
			return accessScopeCmd.PrintYAML()
		},
	}

	cmd.Flags().StringVar(&accessScopeCmd.accessScope.Name, "name", "", "name of the access scope")
	cmd.Flags().StringVar(&accessScopeCmd.accessScope.Description, "description", "",
		"description of the access scope")

	cmd.Flags().Var(&includedObjectsFlag{includedObjects: &accessScopeCmd.accessScope.Rules.IncludedObjects}, "included",
		`list of clusters and their namespaces that should be included within the access scope.
In case all namespaces of a specific cluster should be included, specify --included cluster-name.
In case only a subset of namespace should be included, specify --included cluster-name=namespaceA,namespaceB`)

	// Currently, its only support to provide a single cluster-label-selector for the access scope.
	// The reason is of the complexity of the resulting struct, its currently not possible to associated N requirements
	// with M label selectors (cluster or namespace). Hence, the command line option only currently allows to create
	// a single one, with the hint that advanced users may adjust the output YAML to include additional label selectors
	// if they wish to do so.

	cmd.Flags().Var(&requirementFlag{requirements: &accessScopeCmd.clusterRequirements}, "cluster-label-selector",
		fmt.Sprintf(labelSelectorUsage, "cluster-label-selector", "cluster-label-selector"))

	cmd.Flags().Var(&requirementFlag{requirements: &accessScopeCmd.namespaceRequirements}, "namespace-label-selector",
		fmt.Sprintf(labelSelectorUsage, "namespace-label-selector", "namespace-label-selector"))

	utils.Must(cmd.MarkFlagRequired("name"))

	return cmd
}

type accessScopeCmd struct {
	accessScope *declarativeconfig.AccessScope
	env         environment.Environment

	clusterRequirements   []declarativeconfig.Requirement
	namespaceRequirements []declarativeconfig.Requirement

	configMap string
	secret    string
	namespace string
}

func (a *accessScopeCmd) Construct(cmd *cobra.Command) error {
	configMap, secret, namespace, err := k8sobject.ReadK8sObjectFlags(cmd)
	if err != nil {
		return errors.Wrap(err, "reading config map flag values")
	}
	a.configMap = configMap
	a.namespace = namespace
	a.secret = secret
	return nil
}

func (a *accessScopeCmd) Validate() error {
	if len(a.clusterRequirements) > 0 {
		a.accessScope.Rules.ClusterLabelSelectors = []declarativeconfig.LabelSelector{
			{Requirements: a.clusterRequirements},
		}
	}
	if len(a.namespaceRequirements) > 0 {
		a.accessScope.Rules.NamespaceLabelSelectors = []declarativeconfig.LabelSelector{
			{Requirements: a.namespaceRequirements},
		}
	}

	t := transform.New()
	protos, err := t.Transform(a.accessScope)
	if err != nil {
		return errors.Wrap(err, "transforming access scope")
	}
	accessScopeType := reflect.TypeOf((*storage.SimpleAccessScope)(nil))
	if len(protos[accessScopeType]) != 1 {
		return errox.InvalidArgs.Newf("expected a single access scope after transformation, but was %d",
			len(protos[accessScopeType]))
	}
	accessScope := protos[accessScopeType][0].(*storage.SimpleAccessScope)
	if err := accessscope.ValidateSimpleAccessScopeProto(accessScope); err != nil {
		return errors.Wrap(err, "validating access scope")
	}
	return nil
}

func (a *accessScopeCmd) PrintYAML() error {
	yamlOut := &bytes.Buffer{}
	enc := yaml.NewEncoder(yamlOut)
	if err := enc.Encode(a.accessScope); err != nil {
		return errors.Wrap(err, "creating the YAML output")
	}
	if err := lint.Lint(yamlOut.Bytes()); err != nil {
		return errors.Wrap(err, "linting the YAML output")
	}
	if a.configMap != "" || a.secret != "" {
		return errors.Wrap(k8sobject.WriteToK8sObject(context.Background(), a.configMap, a.secret, a.namespace,
			fmt.Sprintf("%s-%s", a.accessScope.Type(), a.accessScope.Name), yamlOut.Bytes()),
			"writing the YAML output to config map")
	}

	if _, err := a.env.InputOutput().Out().Write(yamlOut.Bytes()); err != nil {
		return errors.Wrap(err, "writing the YAML output")
	}
	return nil
}

// Implementation of pflag.Value to support complex object declarativeconfig.IncludedObject.
type includedObjectsFlag struct {
	includedObjects *[]declarativeconfig.IncludedObject
}

func (i *includedObjectsFlag) String() string {
	var res []string

	for _, obj := range *i.includedObjects {
		res = append(res, obj.Cluster+"="+strings.Join(obj.Namespaces, ","))
	}

	s, _ := json.Marshal(res)

	return "[" + string(s) + "]"
}

func (i *includedObjectsFlag) Set(v string) error {
	c := strings.Count(v, "=")
	switch c {
	case 0:
		*i.includedObjects = append(*i.includedObjects, declarativeconfig.IncludedObject{Cluster: v})
	case 1:
		keyValuePair := strings.SplitN(v, "=", 2)
		*i.includedObjects = append(*i.includedObjects, declarativeconfig.IncludedObject{
			Cluster:    keyValuePair[0],
			Namespaces: strings.Split(keyValuePair[1], ","),
		})
	default:
		return fmt.Errorf("%s must be either formatted as key or as key=value pair", v)
	}
	return nil
}

func (i *includedObjectsFlag) Type() string {
	return "included-object"
}

// Implementation of pflag.Value to support complex object declarativeconfig.Requirement.
type requirementFlag struct {
	requirements *[]declarativeconfig.Requirement
}

func (r *requirementFlag) String() string {
	res := make([]string, 0, len(*r.requirements))

	for _, requirement := range *r.requirements {
		requirementString := fmt.Sprintf("key=%q;operator=%q", requirement.Key,
			storage.SetBasedLabelSelector_Operator(requirement.Operator))
		if len(requirement.Values) != 0 {
			requirementString = fmt.Sprintf("%s;values=%q",
				requirementString, strings.Join(requirement.Values, ","))
		}
		res = append(res, requirementString)
	}

	s, _ := json.Marshal(res)

	return "[" + string(s) + "]"
}

func (r *requirementFlag) Set(v string) error {
	requirement, err := retrieveRequirement(v)
	if err != nil {
		return err
	}
	*r.requirements = append(*r.requirements, *requirement)
	return nil
}

func (r *requirementFlag) Type() string {
	return "requirement"
}

func retrieveRequirement(s string) (*declarativeconfig.Requirement, error) {
	c := strings.Count(s, ";")
	if c != 1 && c != 2 {
		return nil, fmt.Errorf("%s must either be formatted as key=v;operator=v or key=v;operator=v;values=v", s)
	}

	kvPairs := strings.Split(s, ";")

	requirement := &declarativeconfig.Requirement{}
	for _, kvPair := range kvPairs {
		if strings.Count(kvPair, "=") != 1 {
			return nil, fmt.Errorf("%s must specify key=value", kvPair)
		}

		kv := strings.Split(kvPair, "=")

		switch kv[0] {
		case "key":
			requirement.Key = kv[1]
		case "operator":
			op, ok := storage.SetBasedLabelSelector_Operator_value[kv[1]]
			if !ok {
				return nil, fmt.Errorf("operator %s must be one of the allowed values: [%s]", kvPair,
					strings.Join(maputil.Keys(storage.SetBasedLabelSelector_Operator_value), ","))
			}
			requirement.Operator = declarativeconfig.Operator(op)
		case "values":
			requirement.Values = strings.Split(kv[1], ",")
		default:
			return nil, fmt.Errorf("%s must specify either key, operator, values", kvPair)
		}
	}

	return requirement, nil
}
