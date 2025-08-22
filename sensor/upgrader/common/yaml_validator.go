package common

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kubectl/pkg/validation"
)

type yamlValidator struct {
	jsonValidator validation.Schema
}

func (v yamlValidator) ValidateBytes(data []byte) error {
	jsonData, err := yaml.ToJSON(data)
	if err != nil {
		return errors.Wrap(err, "converting YAML to JSON")
	}
	if err := v.jsonValidator.ValidateBytes(jsonData); err != nil {
		return errors.Wrap(err, "validating YAML JSON schema")
	}
	return nil
}
