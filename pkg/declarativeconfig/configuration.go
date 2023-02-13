package declarativeconfig

import (
	"bytes"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// ConfigurationType specifies the type of declarative configuration.
type ConfigurationType = string

// The list of currently supported and implemented declarative configuration types.
const (
	AuthProviderConfiguration  ConfigurationType = "auth-provider"
	AccessScopeConfiguration   ConfigurationType = "access-scope"
	PermissionSetConfiguration ConfigurationType = "permission-set"
	RoleConfiguration          ConfigurationType = "role"
)

// Configuration specifies a declarative configuration.
type Configuration interface {
	Type() ConfigurationType
}

// ConfigurationFromRawBytes takes in a list of raw bytes, i.e. file contents, and returns the unmarshalled
// configurations.
// It will return an error if:
//   - the raw bytes are in invalid format, i.e. not YAML format.
//   - the YAML cannot be unmarshalled into valid configuration type.
func ConfigurationFromRawBytes(rawConfigurations ...[]byte) ([]Configuration, error) {
	var configurations []Configuration
	for _, rawConfiguration := range rawConfigurations {
		var objects []interface{}
		err := yaml.Unmarshal(rawConfiguration, &objects)
		if err == nil {
			configs, err := fromUnstructuredConfigs(objects)
			if err != nil {
				return nil, errors.Wrap(err, "unmarshalling list of raw configuration")
			}
			configurations = append(configurations, configs...)
		} else {
			config, err := fromRawBytes(rawConfiguration)
			if err != nil {
				return nil, errors.Wrap(err, "unmarshalling raw configuration")
			}
			configurations = append(configurations, config)
		}
	}

	return configurations, nil
}

func fromUnstructuredConfigs(unstructuredConfigs []interface{}) ([]Configuration, error) {
	configurations := make([]Configuration, 0, len(unstructuredConfigs))
	// Not sure how to do this otherwise, we essentially have to marshal each configuration and unmarshal it afterwards.
	for _, unstructuredConfig := range unstructuredConfigs {
		rawConfigurationBytes, err := yaml.Marshal(unstructuredConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "marshalling configuration from list %+v", unstructuredConfig)
		}
		config, err := fromRawBytes(rawConfigurationBytes)
		if err != nil {
			return nil, err
		}
		configurations = append(configurations, config)
	}
	return configurations, nil
}

func fromRawBytes(rawConfiguration []byte) (Configuration, error) {

	for _, configurationType := range []ConfigurationType{AuthProviderConfiguration, AccessScopeConfiguration,
		PermissionSetConfiguration, RoleConfiguration} {
		switch configurationType {
		case AuthProviderConfiguration:
			var authProvider AuthProvider
			if err := decodeYAMLToConfiguration(rawConfiguration, &authProvider); err != nil {
				break
			}
			return &authProvider, nil
		case AccessScopeConfiguration:
			var accessScope AccessScope
			if err := decodeYAMLToConfiguration(rawConfiguration, &accessScope); err != nil {
				break
			}
			return &accessScope, nil
		case PermissionSetConfiguration:
			var permissionSet PermissionSet
			if err := decodeYAMLToConfiguration(rawConfiguration, &permissionSet); err != nil {
				break
			}
			return &permissionSet, nil
		case RoleConfiguration:
			var role Role
			if err := decodeYAMLToConfiguration(rawConfiguration, &role); err != nil {
				break
			}
			return &role, nil
		}
	}
	return nil, errors.Errorf("raw configuration found that didn't match any of the given configurations: %s",
		rawConfiguration)
}

func decodeYAMLToConfiguration(rawYAML []byte, configuration Configuration) error {
	dec := yaml.NewDecoder(bytes.NewReader(rawYAML))
	dec.KnownFields(true)
	if err := dec.Decode(configuration); err != nil {
		return err
	}
	return nil
}
