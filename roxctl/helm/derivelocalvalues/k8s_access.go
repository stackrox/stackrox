package derivelocalvalues

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/stackrox/rox/pkg/reflectutils"
	"github.com/stackrox/rox/pkg/set"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/util/jsonpath"
)

type k8sObjectDescription struct {
	k8sObjectDescriptionInterface
	warnings []string
}

type k8sObjectDescriptionInterface interface {
	get(ctx context.Context, kind string, name string) (*unstructured.Unstructured, error)
}

func (k *k8sObjectDescription) evaluate(ctx context.Context, kind string, name string, path string) interface{} {
	res, err := k.get(ctx, kind, name)
	if err != nil {
		k.warn("Failed to lookup resource %s/%s: %v", kind, name, err)
		return nil
	}
	return unstructuredLookup(kind, name, *res, path)
}

func (k *k8sObjectDescription) evaluateOrDefault(ctx context.Context, kind string, name string, path string, def interface{}) interface{} {
	res := k.evaluate(ctx, kind, name, path)
	if res == nil {
		res = def
	}
	return res
}

func (k *k8sObjectDescription) evaluateToObject(ctx context.Context, kind string, name string, jsonpath string, def map[string]interface{}) map[string]interface{} {
	var objStrings map[string]interface{}
	x := k.evaluateOrDefault(ctx, kind, name, jsonpath, def)
	switch obj := x.(type) {
	case map[interface{}]interface{}:
		objStrings = make(map[string]interface{})
		for k, v := range obj {
			s, ok := k.(string)
			if !ok {
				continue
			}
			objStrings[s] = v
		}

	case map[string]interface{}:
		objStrings = obj

	default:
		k.warn("Unexpected data type (%T) at JsonPath %q for resource %s/%s: %v", x, jsonpath, kind, name, x)
		return def
	}

	return objStrings
}

func (k *k8sObjectDescription) evaluateToSlice(ctx context.Context, kind string, name string, jsonpath string, def []interface{}) []interface{} {
	x := k.evaluateOrDefault(ctx, kind, name, jsonpath, def)
	slice, ok := x.([]interface{})
	if !ok {
		k.warn("Unexpected data type (%T) at JsonPath %q for resource %s/%s: %v", x, jsonpath, kind, name, x)
		return def
	}
	return slice
}

func (k *k8sObjectDescription) evaluateToSubObject(ctx context.Context, kind string, name string, jsonpath string, retainKeys []string, def map[string]interface{}) map[string]interface{} {
	var objStrings map[string]interface{}
	x := k.evaluate(ctx, kind, name, jsonpath)
	if reflectutils.IsNil(x) {
		return def
	}

	switch obj := x.(type) {
	case map[interface{}]interface{}:
		objStrings = make(map[string]interface{})
		for k, v := range obj {
			s, ok := k.(string)
			if !ok {
				continue
			}
			objStrings[s] = v
		}
	case map[string]interface{}:
		objStrings = obj
	default:
		k.warn("Unexpected data type (%T) at JsonPath %q for resource %s/%s: %v", x, jsonpath, kind, name, x)
		return def
	}

	// Remove any keys from object, which are not in retainKeys.
	retainKeysSet := set.NewStringSet(retainKeys...)
	for objKey := range objStrings {
		if !retainKeysSet.Contains(objKey) {
			delete(objStrings, objKey)
		}
	}

	return objStrings
}

func (k *k8sObjectDescription) evaluateToString(ctx context.Context, kind string, name string, jsonpath string, def string) string {
	x := k.evaluateOrDefault(ctx, kind, name, jsonpath, def)
	s, ok := x.(string)
	if !ok {
		k.warn("Unexpected data type (%T) at JsonPath %q for resource %s/%s: %v", x, jsonpath, kind, name, x)
		return def
	}
	return s
}

func (k *k8sObjectDescription) evaluateToStringSlice(ctx context.Context, kind string, name string,
	jsonpath string, def []string) []string {
	x := k.evaluateOrDefault(ctx, kind, name, jsonpath, def)
	s, ok := x.([]string)
	if !ok {
		k.warn("Unexpected data type (%T) at JsonPath %q for resource %s/%s: %v", x, jsonpath, kind, name, x)
		return def
	}
	return s
}

func (k *k8sObjectDescription) evaluateToStringP(ctx context.Context, kind string, name string, jsonpath string) *string {
	s := k.evaluateToString(ctx, kind, name, jsonpath, "")
	if s == "" {
		return nil
	}
	return &s
}

// This accessor function supports retrieving data from the objects `data` and `stringData`. Base64-decoding
// is applied when required (i.e., when the requested value was found within `data`).
func (k *k8sObjectDescription) lookupSecretStringP(ctx context.Context, name string, field string) *string {
	var secret *string

	if obj := k.evaluateToObject(ctx, "secret", name, "{.stringData}", nil); obj != nil && obj[field] != nil {
		fieldVal, ok := obj[field].(string)
		if !ok {
			k.warn("Unexpected type %T for stringData.%q within the secret %q: %v",
				obj[field], field, name, obj[field])
			return nil
		}
		secret = &fieldVal
	} else if obj := k.evaluateToObject(ctx, "secret", name, "{.data}", nil); obj != nil && obj[field] != nil {
		fieldVal, ok := obj[field].(string)
		if !ok {
			k.warn("Unexpected type %T for data.%q within the secret %q: %v",
				obj[field], field, name, obj[field])
			return nil
		}

		decoded, err := base64.StdEncoding.DecodeString(fieldVal)
		if err != nil {
			k.warn("Failed to base64-decode secret %s/%s: %v", name, field, err)
		}

		decodedStr := string(decoded)
		secret = &decodedStr
	}

	return secret
}

func (k *k8sObjectDescription) evaluateToInt64(ctx context.Context, kind string, name string, jsonpath string, def int64) int64 {
	x := k.evaluateOrDefault(ctx, kind, name, jsonpath, def)
	switch i := x.(type) {
	case int:
		return int64(i)
	case int16:
		return int64(i)
	case int32:
		return int64(i)
	case int64:
		return i
	default:
		k.warn("Unexpected data type (%T) at JsonPath %q for resource %s/%s: %v", x, jsonpath, kind, name, x)
		return def
	}
}

func (k *k8sObjectDescription) Exists(ctx context.Context, kind string, name string) bool {
	_, err := k.get(ctx, kind, name)
	return err == nil
}

func unstructuredLookup(kind string, name string, u unstructured.Unstructured, path string) interface{} {
	jp := jsonpath.New(fmt.Sprintf("unstructured Lookup for %s/%s", kind, name))
	err := jp.Parse(path)
	if err != nil {
		// This is a bug in the jsonpath description itself.
		panic(fmt.Sprintf("Error: Invalid json path %q", path))
	}

	vals, err := jp.FindResults(u.UnstructuredContent())
	if err != nil {
		return nil
	}

	if len(vals) == 0 || len(vals[0]) == 0 {
		return nil
	}
	return vals[0][0].Interface()
}

func newK8sObjectDescription(i k8sObjectDescriptionInterface) k8sObjectDescription {
	return k8sObjectDescription{k8sObjectDescriptionInterface: i, warnings: nil}
}

func (k *k8sObjectDescription) getWarnings() []string {
	return k.warnings
}

func (k *k8sObjectDescription) warn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	k.warnings = append(k.warnings, msg)
}
