package migratetooperator

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// Source provides access to Kubernetes resources for migration detection.
// Deployment returns an error if not found. Service, Secret, and Route
// return (nil, nil) if not found.
type Source interface {
	Deployment(name string) (*appsv1.Deployment, error)
	Service(name string) (*corev1.Service, error)
	Secret(name string) (*corev1.Secret, error)
	Route(name string) (bool, error)
}
