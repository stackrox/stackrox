package migratetooperator

import (
	appsv1 "k8s.io/api/apps/v1"
)

// source provides access to Kubernetes resources from either a directory
// of YAML manifests or a live cluster.
type source interface {
	// CentralDBDeployment returns the central-db Deployment.
	// It returns a non-nil error if the deployment is not found or cannot be retrieved.
	CentralDBDeployment() (*appsv1.Deployment, error)
}
