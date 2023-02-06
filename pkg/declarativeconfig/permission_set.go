package declarativeconfig

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"gopkg.in/yaml.v3"
)

// PermissionSet is representation of storage.PermissionSet that supports transformation from YAML.
type PermissionSet struct {
	Name        string               `yaml:"name,omitempty"`
	Description string               `yaml:"description,omitempty"`
	Resources   []ResourceWithAccess `yaml:"resources,omitempty"`
}

// Type returns the PermissionSetConfiguration type.
func (p *PermissionSet) Type() ConfigurationType {
	return PermissionSetConfiguration
}

// Access is representation of storage.Access that supports transformation from YAML.
type Access storage.Access

// ResourceWithAccess unites resource name and corresponding access level.
type ResourceWithAccess struct {
	Resource string `yaml:"resource,omitempty"`
	Access   Access `yaml:"access,omitempty"`
}

// MarshalYAML transforms Access to YAML format.
func (a Access) MarshalYAML() ([]byte, error) {
	protoAccess := storage.Access(a)
	return []byte(protoAccess.String()), nil
}

// UnmarshalYAML makes transformation from YAML to Access.
func (a *Access) UnmarshalYAML(value *yaml.Node) error {
	var v string
	if err := value.Decode(&v); err != nil {
		return err
	}
	i, ok := storage.Access_value[v]
	if !ok {
		return errors.New("not found")
	}
	*a = Access(i)
	return nil
}
