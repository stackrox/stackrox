package maputil

import "github.com/stackrox/stackrox/pkg/reflectutils"

// NormalizeGenericMap removes empty values from the provided generic map.
func NormalizeGenericMap(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{})
	for k, v := range src {
		if obj, ok := v.(map[string]interface{}); ok {
			v = NormalizeGenericMap(obj)
		}
		if reflectutils.IsNil(v) {
			continue
		}

		dst[k] = v
	}
	if len(dst) == 0 {
		return nil
	}
	return dst
}
