package kubernetes

import (
	"regexp"
)

const (
	// excludedOperatorNamespace defines the constant for openshift-operators which is a default namespace for many
	// third-party operators that we do *not* want to specify as system services
	excludedOperatorNamespace = "openshift-operators"
)

var (
	systemNamespaceRegex = regexp.MustCompile(`^kube.|^openshift.*|^redhat.*|^istio-system$`)
)

// IsSystemNamespace returns whether or not the namespace should be considered a system namespace
func IsSystemNamespace(namespace string) bool {
	if namespace == excludedOperatorNamespace {
		return false
	}
	return systemNamespaceRegex.MatchString(namespace)
}
