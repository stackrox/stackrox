package migratetooperator

import (
	appsv1 "k8s.io/api/apps/v1"
)

// source provides access to Kubernetes resources from either a directory
// of YAML manifests or a live cluster.
type source interface {
	// CentralDBDeployment returns the central-db Deployment, or nil if not found.
	CentralDBDeployment() (*appsv1.Deployment, error)
}
