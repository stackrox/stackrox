package common

import (
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kubectl/pkg/validation"
)

type yamlValidator struct {
	jsonValidator validation.Schema
}

func (v yamlValidator) ValidateBytes(data []byte) error {
	jsonData, err := yaml.ToJSON(data)
	if err != nil {
		return err
	}
	return v.jsonValidator.ValidateBytes(jsonData)
}
