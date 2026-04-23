package migratetooperator

import (
	appsv1 "k8s.io/api/apps/v1"
)

// Source provides access to Kubernetes resources for migration detection.
type Source interface {
	CentralDeployment() (*appsv1.Deployment, error)
	CentralDBDeployment() (*appsv1.Deployment, error)
	ResourceByKindAndName(kind, name string) (bool, map[string]interface{}, error)
}
