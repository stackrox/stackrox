package sac

import (
	"fmt"
	"strings"
)

// ClusterNSScopeString creates a flattened cluster/ns scope string. This is supposed
// to be used in the context of embedding scope information to indexed objects.
func ClusterNSScopeString(clusterID string, namespace string) string {
	if namespace == "" {
		return clusterID
	}
	return fmt.Sprintf("%s:%s", clusterID, namespace)
}

// ClusterNSScopeStringFromObject returns the cluster/ns scope string from the given namespace-scoped
// object.
func ClusterNSScopeStringFromObject(object NamespaceScopedObject) string {
	return ClusterNSScopeString(object.GetClusterId(), object.GetNamespace())
}

// ParseClusterNSScopeString parses a cluster/ns scope string into a ScopeKey slice.
func ParseClusterNSScopeString(str string) []ScopeKey {
	parts := strings.SplitN(str, ":", 2)
	switch {
	case len(parts) == 0:
		return nil
	case len(parts) == 1 || parts[1] == "":
		return []ScopeKey{ClusterScopeKey(parts[0])}
	default:
		return []ScopeKey{ClusterScopeKey(parts[0]), NamespaceScopeKey(parts[1])}
	}
}
