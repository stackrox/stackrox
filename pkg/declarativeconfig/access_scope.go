package declarativeconfig

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"gopkg.in/yaml.v3"
)

// AccessScope is representation of storage.AccessScope that supports transformation from YAML.
type AccessScope struct {
	Name        string `yaml:"name,omitempty"`
	Description string `yaml:"description,omitempty"`
	Rules       Rules  `yaml:"rules,omitempty"`
}

// Type returns the AccessScopeConfiguration type.
func (a *AccessScope) Type() ConfigurationType {
	return AccessScopeConfiguration
}

// Operator is representation of storage.SetBasedLabelSelector_Operator that supports transformation from YAML.
type Operator storage.SetBasedLabelSelector_Operator

// MarshalYAML transforms Operator to YAML format.
func (a Operator) MarshalYAML() ([]byte, error) {
	protoAccess := storage.SetBasedLabelSelector_Operator(a)
	return []byte(protoAccess.String()), nil
}

// UnmarshalYAML makes transformation from YAML to Operator.
func (a *Operator) UnmarshalYAML(value *yaml.Node) error {
	var v string
	if err := value.Decode(&v); err != nil {
		return err
	}
	i, ok := storage.SetBasedLabelSelector_Operator_value[v]
	if !ok {
		return errors.Errorf("Operator value %s not found", v)
	}
	*a = Operator(i)
	return nil
}

// Requirement is representation of storage.SetBasedLabelSelector_Requirement that supports transformation from YAML.
type Requirement struct {
	Key      string   `yaml:"key,omitempty"`
	Operator Operator `yaml:"operator,omitempty"`
	Values   []string `yaml:"values,omitempty"`
}

// LabelSelector is representation of storage.SetBasedLabelSelector that supports transformation from YAML.
type LabelSelector struct {
	Requirements []Requirement `yaml:"requirements,omitempty"`
}

// IncludedObject represents list of included into access scope namespaces within the specified cluster.
// If namespaces list is empty, that means the whole cluster is included into access scope.
type IncludedObject struct {
	Cluster    string   `yaml:"cluster,omitempty"`
	Namespaces []string `yaml:"namespaces,omitempty"`
}

// Rules is representation of storage.SimpleAccessScope_Rules that supports transformation from YAML.
type Rules struct {
	IncludedObjects         []IncludedObject `yaml:"included,omitempty"`
	ClusterLabelSelectors   []LabelSelector  `yaml:"clusterLabelSelectors,omitempty"`
	NamespaceLabelSelectors []LabelSelector  `yaml:"namespaceLabelSelectors,omitempty"`
}
