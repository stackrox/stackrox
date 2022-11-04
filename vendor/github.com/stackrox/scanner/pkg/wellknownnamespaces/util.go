package wellknownnamespaces

import "strings"

// IsAlpineNamespace returns true if the given argument identifies an Alpine namespace.
// The namespace is expected to be of form `namespacename:version` but only `namespacename` is required.
// For example: rhel:7, rhel:8, centos:8, ubuntu:14.04.
func IsAlpineNamespace(namespace string) bool {
	return strings.HasPrefix(namespace, "alpine")
}

// IsCentOSNamespace returns true if the given argument identifies a CentOS namespace.
// The namespace is expected to be of form `namespacename:version` but only `namespacename` is required.
// For example: rhel:7, rhel:8, centos:8, ubuntu:14.04.
func IsCentOSNamespace(namespace string) bool {
	return strings.HasPrefix(namespace, "centos")
}

// IsRHELNamespace returns true if the given argument identifies a RHEL namespace.
// The namespace is expected to be of form `namespacename:version` but only `namespacename` is required.
// For example: rhel:7, rhel:8, centos:8, ubuntu:14.04.
func IsRHELNamespace(namespace string) bool {
	return strings.HasPrefix(namespace, "rhel")
}

// IsUbuntuNamespace returns true if the given argument identifies an Ubuntu namespace.
// The namespace is expected to be of form `namespacename:version` but only `namespacename` is required.
// For example: rhel:7, rhel:8, centos:8, ubuntu:14.04.
func IsUbuntuNamespace(namespace string) bool {
	return strings.HasPrefix(namespace, "ubuntu")
}
