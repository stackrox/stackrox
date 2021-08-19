package test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"reflect"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

// truthiness returns the truthiness value of an arbitary value. The nil interface and zero values are always falsy.
// Empty slices and maps are falsy as well, even if they are non-nil. All other values are truthy.
func truthiness(val interface{}) bool {
	if val == nil {
		return false
	}
	rval := reflect.ValueOf(val)
	if rval.IsZero() {
		return false
	}
	if rval.Kind() == reflect.Slice || rval.Kind() == reflect.Map {
		return rval.Len() > 0
	}
	return true
}

// unmarshalYamlFromFileStrict unmarshals the contents of filename into out, relying on YAML-to-JSON semantics (i.e.,
// honoring `json:"..."` tags instead of requiring `yaml:"..."` tags). Any field that is not present in the output data
// type, as well as any duplicate keys within the same YAML object will result in an error.
func unmarshalYamlFromFileStrict(filename string, out interface{}) error {
	contents, err := ioutil.ReadFile(filename)
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
