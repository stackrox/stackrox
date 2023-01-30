package declarativeconfig

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"gopkg.in/yaml.v3"
)

type AccessScope struct {
	Name        string `yaml:"name,omitempty"`
	Description string `yaml:"description,omitempty"`
	Rules       Rules  `yaml:"rules,omitempty"`
}

type Namespace struct {
	Cluster   string `yaml:"cluster,omitempty"`
	Namespace string `yaml:"namespace,omitempty"`
}

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

type Requirement struct {
	Key      string   `yaml:"key,omitempty"`
	Operator Operator `yaml:"operator,omitempty"`
	Values   []string `yaml:"values,omitempty"`
}

type LabelSelector struct {
	Requirements []Requirement `yaml:"requirements,omitempty"`
}

type Rules struct {
	IncludedClusters        []string        `yaml:"includedClusters,omitempty"`
	IncludedNamespaces      []Namespace     `yaml:"includedNamespaces,omitempty"`
	ClusterLabelSelectors   []LabelSelector `yaml:"clusterLabelSelectors,omitempty"`
	NamespaceLabelSelectors []LabelSelector `yaml:"namespaceLabelSelectors,omitempty"`
}
