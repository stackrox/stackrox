package dynamic

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
)

const (
	// MaxClusterNameLength is the maximum allowed length for a cluster name.
	// Based on Kubernetes naming conventions.
	MaxClusterNameLength = 253

	// MaxNamespaceNameLength is the maximum allowed length for a namespace name.
	// Kubernetes limit is 63 characters.
	MaxNamespaceNameLength = 63

	// MaxDeploymentNameLength is the maximum allowed length for a deployment name.
	// Kubernetes limit is 253 characters.
	MaxDeploymentNameLength = 253
)

var (
	// k8sNameRegex validates Kubernetes resource names (RFC 1123 DNS label format).
	// Must consist of lower case alphanumeric characters, '-' or '.', and must
	// start and end with an alphanumeric character.
	k8sNameRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)
)

// BuildDynamicScope creates a DynamicAccessScope for embedding in token claims.
// It validates the provided cluster, namespace, and deployment names according
// to Kubernetes naming conventions.
//
// Parameters:
//   - clusterName: Required. Must be a valid cluster identifier.
//   - namespace: Optional. If empty, scope grants access to all namespaces in cluster.
//   - deployment: Optional. Only valid if namespace is specified. If empty, grants
//     access to all deployments in the namespace.
//
// Returns an error if:
//   - clusterName is empty
//   - namespace or deployment names exceed Kubernetes length limits
//   - names contain invalid characters (must follow DNS label format)
//   - deployment is specified without namespace
func BuildDynamicScope(clusterName, namespace, deployment string) (*storage.DynamicAccessScope, error) {
	// Cluster name is mandatory
	if clusterName == "" {
		return nil, errox.InvalidArgs.New("cluster name is required")
	}

	if len(clusterName) > MaxClusterNameLength {
		return nil, errox.InvalidArgs.Newf("cluster name exceeds maximum length of %d", MaxClusterNameLength)
	}

	// Note: Cluster names in StackRox may not strictly follow k8s naming conventions
	// (e.g., they might be UUIDs with uppercase), so we don't validate them with regex

	// Validate namespace if provided
	if namespace != "" {
		if len(namespace) > MaxNamespaceNameLength {
			return nil, errox.InvalidArgs.Newf("namespace name exceeds maximum length of %d", MaxNamespaceNameLength)
		}
		if !k8sNameRegex.MatchString(namespace) {
			return nil, errox.InvalidArgs.Newf("namespace name %q is not a valid Kubernetes resource name", namespace)
		}
	}

	// Validate deployment if provided
	if deployment != "" {
		if namespace == "" {
			return nil, errox.InvalidArgs.New("deployment scope requires namespace to be specified")
		}
		if len(deployment) > MaxDeploymentNameLength {
			return nil, errox.InvalidArgs.Newf("deployment name exceeds maximum length of %d", MaxDeploymentNameLength)
		}
		if !k8sNameRegex.MatchString(deployment) {
			return nil, errox.InvalidArgs.Newf("deployment name %q is not a valid Kubernetes resource name", deployment)
		}
	}

	return &storage.DynamicAccessScope{
		ClusterName: clusterName,
		Namespace:   namespace,
		Deployment:  deployment,
	}, nil
}

// ScopeDescription returns a human-readable description of the scope's extent.
func ScopeDescription(scope *storage.DynamicAccessScope) string {
	if scope == nil {
		return "unrestricted"
	}

	if scope.Deployment != "" {
		return fmt.Sprintf("cluster=%s, namespace=%s, deployment=%s",
			scope.ClusterName, scope.Namespace, scope.Deployment)
	}

	if scope.Namespace != "" {
		return fmt.Sprintf("cluster=%s, namespace=%s (all deployments)",
			scope.ClusterName, scope.Namespace)
	}

	return fmt.Sprintf("cluster=%s (all namespaces)", scope.ClusterName)
}

// IsClusterScoped returns true if the scope grants access to the entire cluster
// (i.e., no namespace/deployment restrictions).
func IsClusterScoped(scope *storage.DynamicAccessScope) bool {
	return scope != nil && scope.Namespace == "" && scope.Deployment == ""
}

// IsNamespaceScoped returns true if the scope grants access to a specific
// namespace but all deployments within it.
func IsNamespaceScoped(scope *storage.DynamicAccessScope) bool {
	return scope != nil && scope.Namespace != "" && scope.Deployment == ""
}

// IsDeploymentScoped returns true if the scope grants access to a specific deployment.
func IsDeploymentScoped(scope *storage.DynamicAccessScope) bool {
	return scope != nil && scope.Deployment != ""
}
