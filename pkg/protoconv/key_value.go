package protoconv

import (
	"sort"

	"github.com/stackrox/rox/generated/api/v1"
)

// ConvertDeploymentKeyValueMap converts from a map[string]string to the proto deployment label
func ConvertDeploymentKeyValueMap(keyValueMap map[string]string) []*v1.Deployment_KeyValue {
	labels := make([]*v1.Deployment_KeyValue, 0, len(keyValueMap))
	for k, v := range keyValueMap {
		labels = append(labels, &v1.Deployment_KeyValue{Key: k, Value: v})
	}
	sort.SliceStable(labels, func(i, j int) bool {
		return labels[i].GetKey() < labels[j].GetKey()
	})
	return labels
}

// ConvertDeploymentKeyValues converts from a proto deployment key value to map
func ConvertDeploymentKeyValues(keyValues []*v1.Deployment_KeyValue) map[string]string {
	m := make(map[string]string, len(keyValues))
	for _, keyValue := range keyValues {
		m[keyValue.GetKey()] = keyValue.GetValue()
	}
	return m
}
