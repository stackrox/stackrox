package user

import (
	"sort"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
)

// ConvertAttributes converts a map of user attributes to v1.UserAttribute
func ConvertAttributes(attrMap map[string][]string) []*v1.UserAttribute {
	if attrMap == nil {
		return nil
	}

	result := make([]*v1.UserAttribute, 0, len(attrMap))
	for k, vs := range attrMap {
		attr := &v1.UserAttribute{
			Key:    k,
			Values: vs,
		}
		result = append(result, attr)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Key < result[j].Key
	})
	return result
}
