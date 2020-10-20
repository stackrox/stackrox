package derivelocalvalues

import "github.com/stackrox/rox/pkg/set"

func filterMap(m map[string]interface{}, keysToDelete []string) map[string]interface{} {
	if m == nil {
		return nil
	}
	setKeysToDelete := set.NewStringSet(keysToDelete...)
	mReduced := make(map[string]interface{})
	for k, v := range m {
		if !setKeysToDelete.Contains(k) {
			mReduced[k] = v
		}
	}
	if len(mReduced) == 0 {
		return nil
	}
	return mReduced
}

func normalizeMap(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{})
	for k, v := range src {
		if obj, ok := v.(map[string]interface{}); ok {
			v = normalizeMap(obj)
		}
		if isNil(v) {
			continue
		}

		dst[k] = v
	}
	if len(dst) == 0 {
		return nil
	}
	return dst
}

func envVarSliceToObj(slice []interface{}) map[string]interface{} {
	newObj := make(map[string]interface{})

	for _, x := range slice {
		obj, ok := x.(map[interface{}]interface{})
		if !ok {
			continue
		}
		name := obj["name"]
		if name == nil {
			continue
		}
		nameStr, ok := name.(string)
		if !ok {
			continue
		}
		value := obj["value"]
		if value != nil {
			newObj[nameStr] = value
		}
	}

	if len(newObj) == 0 {
		return nil
	}

	return newObj
}
