package declarativeconfig

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/maputil"
	"gopkg.in/yaml.v3"
)

// PermissionSet is representation of storage.PermissionSet that supports transformation from YAML.
type PermissionSet struct {
	Name        string               `yaml:"name,omitempty"`
	Description string               `yaml:"description,omitempty"`
	Resources   []ResourceWithAccess `yaml:"resources,omitempty"`
}

// ConfigurationType returns the PermissionSetConfiguration type.
func (p *PermissionSet) ConfigurationType() ConfigurationType {
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
func (a Access) MarshalYAML() (interface{}, error) {
	protoAccess := storage.Access(a)
	return protoAccess.String(), nil
}

// UnmarshalYAML makes transformation from YAML to Access.
func (a *Access) UnmarshalYAML(value *yaml.Node) error {
	var v string
	if err := value.Decode(&v); err != nil {
		return err
	}
	i, ok := storage.Access_value[v]
	if !ok {
		return errox.InvalidArgs.Newf("access %s is invalid, valid values are: [%s]", v, strings.Join(
			maputil.Keys(storage.Access_value), ","))
	}
	*a = Access(i)
	return nil
}
