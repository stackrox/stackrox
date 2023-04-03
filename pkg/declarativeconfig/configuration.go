package declarativeconfig

import (
	"bytes"
	"io"

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
		configs, err := parseToConfiguration(rawConfiguration)
		if err != nil {
			return nil, err
		}
		configurations = append(configurations, configs...)
	}

	return configurations, nil
}

func fromUnstructuredConfigs(unstructuredConfigs []interface{}) ([]Configuration, error) {
	configurations := make([]Configuration, 0, len(unstructuredConfigs))
	for _, unstructuredConfig := range unstructuredConfigs {
		config, err := fromUnstructured(unstructuredConfig)
		if err != nil {
			return nil, err
		}
		configurations = append(configurations, config)
	}
	return configurations, nil
}

func fromUnstructured(unstructured interface{}) (Configuration, error) {
	rawConfiguration, err := yaml.Marshal(unstructured)
	if err != nil {
		return nil, errors.Wrap(err, "marshalling unstructured configuration")
	}

	var decodeErrs *multierror.Error

	var authProvider AuthProvider
	err = decodeYAMLToConfiguration(rawConfiguration, &authProvider)
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
	return nil, errors.Wrap(decodeErrs, "unable to unmarshal the configuration")
}

func decodeYAMLToConfiguration(rawYAML []byte, configuration Configuration) error {
	dec := yaml.NewDecoder(bytes.NewReader(rawYAML))
	dec.KnownFields(true)
	if err := dec.Decode(configuration); err != nil {
		return err
	}
	return nil
}

func parseToConfiguration(contents []byte) ([]Configuration, error) {
	dec := yaml.NewDecoder(bytes.NewReader(contents))
	var unstructuredObjs []interface{}
	for {
		var obj interface{}
		err := dec.Decode(&obj)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "decoding YAML file contents")
		}
		unstructuredObjs = append(unstructuredObjs, obj)
	}

	var configurations []Configuration
	for _, unstructured := range unstructuredObjs {
		// Special case: a list of objects.
		listOfObj, ok := unstructured.([]interface{})
		if ok {
			configs, err := fromUnstructuredConfigs(listOfObj)
			if err != nil {
				return nil, err
			}
			configurations = append(configurations, configs...)
			continue
		}

		config, err := fromUnstructured(unstructured)
		if err != nil {
			return nil, err
		}
		configurations = append(configurations, config)
	}

	return configurations, nil
}
