package migratetooperator

import (
	appsv1 "k8s.io/api/apps/v1"
)

// source provides access to Kubernetes resources from either a directory
// of YAML manifests or a live cluster.
type source interface {
	// CentralDeployment returns the central Deployment.
	// It returns a non-nil error if the deployment is not found or cannot be retrieved.
	CentralDeployment() (*appsv1.Deployment, error)

	// CentralDBDeployment returns the central-db Deployment.
	// It returns a non-nil error if the deployment is not found or cannot be retrieved.
	CentralDBDeployment() (*appsv1.Deployment, error)

	// ResourceByKindAndName looks for a resource by kind and metadata.name.
	// Returns (true, data) if found, (false, nil) if not found.
	// The data map contains the raw parsed YAML for further inspection.
	ResourceByKindAndName(kind, name string) (bool, map[string]interface{}, error)
}
