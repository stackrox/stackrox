package migratetooperator

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// Source provides access to Kubernetes resources for migration detection.
type Source interface {
	Deployment(name string) (*appsv1.Deployment, error)
	Service(name string) (*corev1.Service, bool, error)
	Secret(name string) (bool, error)
	Route(name string) (bool, error)
}
