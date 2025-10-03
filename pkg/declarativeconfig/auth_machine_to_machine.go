package declarativeconfig

import (
	"maps"
	"slices"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"go.yaml.in/yaml/v3"
)

// MachineToMachineRoleMapping represents the role attribution part
// of a machine to machine auth configuration.
type MachineToMachineRoleMapping struct {
	Key             string `yaml:"key,omitempty"`
	ValueExpression string `yaml:"value,omitempty"`
	Role            string `yaml:"role,omitempty"`
}

// AuthMachineToMachineConfig represents a machine to machine auth configuration.
type AuthMachineToMachineConfig struct {
	Type                    AuthMachineToMachineConfigType `yaml:"type,omitempty"`
	TokenExpirationDuration string                         `yaml:"tokenExpirationDuration,omitempty"`
	Mappings                []MachineToMachineRoleMapping  `yaml:"mappings,omitempty"`
	Issuer                  string                         `yaml:"issuer,omitempty"`
}

// AuthMachineToMachineConfigType is representation of storage.AuthMachineToMachineConfig_Type
// that supports transformation from YAML.
type AuthMachineToMachineConfigType storage.AuthMachineToMachineConfig_Type

// MarshalYAML transforms AuthMachineToMachineConfigType to YAML format.
func (t AuthMachineToMachineConfigType) MarshalYAML() (interface{}, error) {
	protoType := storage.AuthMachineToMachineConfig_Type(t)
	return protoType.String(), nil
}

// UnmarshalYAML makes transformation from YAML to AuthMachineToMachineConfigType.
func (t *AuthMachineToMachineConfigType) UnmarshalYAML(value *yaml.Node) error {
	var v string
	if err := value.Decode(&v); err != nil {
		return err
	}
	i, ok := storage.AuthMachineToMachineConfig_Type_value[v]
	if !ok {
		return errox.InvalidArgs.Newf("type %s is invalid, valid types are: [%s]", v, strings.Join(
			slices.Collect(maps.Keys(storage.AuthMachineToMachineConfig_Type_value)), ","))
	}
	*t = AuthMachineToMachineConfigType(i)
	return nil
}

func (c *AuthMachineToMachineConfig) ConfigurationType() ConfigurationType {
	return AuthMachineToMachineConfiguration
}
