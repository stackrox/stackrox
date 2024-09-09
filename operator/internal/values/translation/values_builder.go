package translation

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/chartutil"
	v1 "k8s.io/api/core/v1"
)

// ValuesBuilder helps assemble a values map in slightly less verbose way than otherwise with plain maps and errors.
type ValuesBuilder struct {
	data   map[string]interface{}
	errors *multierror.Error
}

// NewValuesBuilder creates and returns new ValuesBuilder instance.
func NewValuesBuilder() ValuesBuilder {
	return ValuesBuilder{}
}

// Build unwraps ValuesBuilder and returns contained map or error.
// Normally Build should only be called once per constructed values graph to get the eventual results.
func (v *ValuesBuilder) Build() (chartutil.Values, error) {
	if v == nil {
		return chartutil.Values{}, nil
	}
	if v.errors != nil && v.errors.Len() != 0 {
		return nil, v.errors
	}

	return ToHelmValues(v.getData())
}

// getData allows deferring allocation of ValuesBuilder.data map until it is actually needed.
func (v *ValuesBuilder) getData() map[string]interface{} {
	if v.data == nil {
		v.data = map[string]interface{}{}
	}
	return v.data
}

// SetPathValue sets a value into its path. It parses a path like "root.child" and creates the necessary child maps for it.
// If the last element already existed it can't be overwritten and records an error.
func (v *ValuesBuilder) SetPathValue(path string, value interface{}) {
	v.setValueByPath([]string{}, v.getData(), strings.Split(path, "."), value)
}

func (v *ValuesBuilder) setValueByPath(visited []string, data map[string]interface{}, path []string, value interface{}) {
	if len(path) == 1 {
		data[path[0]] = value
		return
	}

	visited = append(visited, path[0])
	if _, ok := data[path[0]].(map[string]interface{}); !ok {
		if data[path[0]] == nil {
			data[path[0]] = map[string]interface{}{}
		} else {
			v.SetError(errors.Errorf("Could not overwrite key at path: %v", strings.Join(visited, ".")))
			return
		}
	}

	v.setValueByPath(visited, data[path[0]].(map[string]interface{}), path[1:], value)
}

// SetError appends error(s) to ValuesBuilder errors collection and returns the same ValuesBuilder.
// SetError accumulates errors and allows working with the same ValuesBuilder until ValuesBuilder.Build() is called
// which returns all accumulated errors.
func (v *ValuesBuilder) SetError(err error) *ValuesBuilder {
	v.errors = multierror.Append(v.errors, err)
	return v
}

// AddAllFrom merges key-values from other ValuesBuilder into the given one. It also merges errors, if any.
// AddAllFrom records errors when attempting to overwrite existing keys.
func (v *ValuesBuilder) AddAllFrom(other *ValuesBuilder) *ValuesBuilder {
	if other == nil {
		return v
	}
	if other.errors != nil && other.errors.Len() > 0 {
		v.SetError(other.errors)
		return v
	}
	for key, val := range other.data {
		if v.validateKey(key) == nil {
			v.getData()[key] = val
		}
	}
	return v
}

// AddChild adds values from child ValuesBuilder, if present, to the given one under the specified key. It also merges errors, if any.
// AddChild records an error on attempt to overwrite existing key.
// Important: don't expect child changes made after AddChild call to be reflected on the parent. I.e. AddChild should be
// the last thing that happens in the lifetime of the child ValuesBuilder.
func (v *ValuesBuilder) AddChild(key string, child *ValuesBuilder) *ValuesBuilder {
	if child == nil {
		return v
	}
	if child.errors != nil && child.errors.Len() > 0 {
		v.SetError(child.errors)
		return v
	}
	if len(child.data) == 0 || v.validateKey(key) != nil {
		return v
	}
	v.getData()[key] = child.getData()
	return v
}

// validateKey remembers and returns an error if the key is empty string or the key already exists in contained data.
func (v *ValuesBuilder) validateKey(key string) error {
	if key == "" {
		err := fmt.Errorf("internal error: attempt to set empty key %q", key)
		v.SetError(err)
		return err
	}
	if _, ok := v.data[key]; ok {
		err := fmt.Errorf("internal error: attempt to overwrite existing key %q", key)
		v.SetError(err)
		return err
	}
	return nil
}

// Typed value setters follow.
// Note: if setter for some type is missing, please add it.
// Do not create SetAny(key string, value interface{}) method because its use may lead to unwanted errors in the calling
// code, e.g. accidentally passing a function closure as a value.

// SetBool adds bool value, if present, under the given key. Records error on attempt to overwrite key.
func (v *ValuesBuilder) SetBool(key string, value *bool) {
	if value == nil || v.validateKey(key) != nil {
		return
	}
	v.getData()[key] = *value
}

// SetBoolValue adds bool value under the given key. Records error on attempt to overwrite key.
func (v *ValuesBuilder) SetBoolValue(key string, value bool) {
	if v.validateKey(key) != nil {
		return
	}
	v.getData()[key] = value
}

// SetInt32 adds int32 value, if present, under the given key.  Records error on attempt to overwrite key.
func (v *ValuesBuilder) SetInt32(key string, value *int32) {
	if value == nil || v.validateKey(key) != nil {
		return
	}
	v.getData()[key] = *value
}

// SetString adds string value, if present, under the given key. Records error on attempt to overwrite key.
func (v *ValuesBuilder) SetString(key string, value *string) {
	if value == nil || v.validateKey(key) != nil {
		return
	}
	v.getData()[key] = *value
}

// SetStringValue adds string value under the given key. Records error on attempt to overwrite key.
func (v *ValuesBuilder) SetStringValue(key string, value string) {
	if v.validateKey(key) != nil {
		return
	}
	v.getData()[key] = value
}

// SetPullPolicy adds pull policy value, if present, under the given key. Records error on attempt to overwrite key.
func (v *ValuesBuilder) SetPullPolicy(key string, value *v1.PullPolicy) {
	v.SetString(key, (*string)(value))
}

// SetStringSlice adds slice, if not empty, under the given key. Records error on attempt to overwrite key.
func (v *ValuesBuilder) SetStringSlice(key string, value []string) {
	if len(value) == 0 || v.validateKey(key) != nil {
		return
	}
	v.getData()[key] = value
}

// SetStringMap adds map, if not empty, under the given key. Records error on attempt to overwrite key.
func (v *ValuesBuilder) SetStringMap(key string, value map[string]string) {
	if len(value) == 0 || v.validateKey(key) != nil {
		return
	}
	v.getData()[key] = value
}

// SetResourceList adds resource list value, if not empty, under the given key. Records error on attempt to overwrite key.
func (v *ValuesBuilder) SetResourceList(key string, value v1.ResourceList) {
	if len(value) == 0 || v.validateKey(key) != nil {
		return
	}
	v.getData()[key] = value
}

// SetMap adds values, if not empty, under the given key. Records error on attempt to overwrite key.
func (v *ValuesBuilder) SetMap(key string, values map[string]interface{}) {
	if len(values) == 0 || v.validateKey(key) != nil {
		return
	}
	v.getData()[key] = values
}

// SetMapSlice adds values slice, if not empty, under the given key. Records error on attempt to overwrite key.
func (v *ValuesBuilder) SetMapSlice(key string, values []map[string]interface{}) {
	if len(values) == 0 || v.validateKey(key) != nil {
		return
	}
	v.getData()[key] = values
}

// SetSlice adds values slice, if not empty, under the given key. Records error on attempt to overwrite key.
func (v *ValuesBuilder) SetSlice(key string, values []interface{}) {
	if len(values) == 0 || v.validateKey(key) != nil {
		return
	}
	v.getData()[key] = values
}
