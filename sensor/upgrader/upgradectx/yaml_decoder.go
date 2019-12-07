package upgradectx

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kubectl/pkg/validation"
)

type yamlDecoder struct {
	jsonDecoder runtime.Decoder
}

func (d yamlDecoder) Decode(data []byte, defaults *schema.GroupVersionKind, into runtime.Object) (runtime.Object, *schema.GroupVersionKind, error) {
	jsonData, err := yaml.ToJSON(data)
	if err != nil {
		return nil, nil, err
	}
	return d.jsonDecoder.Decode(jsonData, defaults, into)
}

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
