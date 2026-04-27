package migratetooperator

import (
	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Source provides access to Kubernetes resources for migration detection.
// All methods return (nil, nil) if the resource is not found.
type Source interface {
	Deployment(name string) (*appsv1.Deployment, error)
	DaemonSet(name string) (*appsv1.DaemonSet, error)
	Service(name string) (*corev1.Service, error)
	Secret(name string) (*corev1.Secret, error)
	Route(name string) (*unstructured.Unstructured, error)
	ValidatingWebhookConfiguration(name string) (*admissionv1.ValidatingWebhookConfiguration, error)
}
