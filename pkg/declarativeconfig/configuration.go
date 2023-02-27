package declarativeconfig

import (
	"bytes"

	"github.com/hashicorp/go-multierror"
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
		// A declarative configuration file can either contain a single declarative configuration, or an array of
		// configurations, hence we first check whether we have an array of objects present.
		var objects []interface{}
		err := yaml.Unmarshal(rawConfiguration, &objects)
		if err == nil {
			configs, err := fromUnstructuredConfigs(objects)
			if err != nil {
				return nil, errors.Wrap(err, "unmarshalling list of raw configurations")
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
	for i, unstructuredConfig := range unstructuredConfigs {
		rawConfigurationBytes, err := yaml.Marshal(unstructuredConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "marshalling configuration[%d] from list %+v", i, unstructuredConfig)
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
	var decodeErrs *multierror.Error

	var authProvider AuthProvider
	err := decodeYAMLToConfiguration(rawConfiguration, &authProvider)
	if err == nil {
		return &authProvider, nil
	}
	decodeErrs = multierror.Append(decodeErrs, err)

	var accessScope AccessScope
	err = decodeYAMLToConfiguration(rawConfiguration, &accessScope)
	if err == nil {
		return &accessScope, nil
	}
	decodeErrs = multierror.Append(decodeErrs, err)

	var permissionSet PermissionSet
	err = decodeYAMLToConfiguration(rawConfiguration, &permissionSet)
	if err == nil {
		return &permissionSet, nil
	}
	decodeErrs = multierror.Append(decodeErrs, err)

	var role Role
	err = decodeYAMLToConfiguration(rawConfiguration, &role)
	if err == nil {
		return &role, nil
	}
	decodeErrs = multierror.Append(decodeErrs, err)
	return nil, errors.Wrapf(decodeErrs, "raw configuration %s didn't match any of the given configurations", rawConfiguration)
}

func decodeYAMLToConfiguration(rawYAML []byte, configuration Configuration) error {
	dec := yaml.NewDecoder(bytes.NewReader(rawYAML))
	dec.KnownFields(true)
	if err := dec.Decode(configuration); err != nil {
		return err
	}
	return nil
}
