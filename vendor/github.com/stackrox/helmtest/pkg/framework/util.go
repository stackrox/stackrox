package framework

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

// unmarshalYamlFromFileStrict unmarshals the contents of filename into out, relying on YAML-to-JSON semantics (i.e.,
// honoring `json:"..."` tags instead of requiring `yaml:"..."` tags). Any field that is not present in the output data
// type, as well as any duplicate keys within the same YAML object, will result in an error.
func unmarshalYamlFromFileStrict(filename string, out interface{}) error {
	contents, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	jsonContents, err := yaml.YAMLToJSONStrict(contents)
	if err != nil {
		return errors.Wrapf(err, "converting YAML in file %s to JSON", filename)
	}
	jsonDec := json.NewDecoder(bytes.NewReader(jsonContents))
	jsonDec.DisallowUnknownFields()
	if err := jsonDec.Decode(out); err != nil {
		return errors.Wrapf(err, "decoding YAML in file %s", filename)
	}
	return nil
}
