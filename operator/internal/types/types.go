package types

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SecretDataMap represents data stored as part of a secret.
type SecretDataMap map[string][]byte

// K8sObject represents a Kubernetes object
type K8sObject interface {
	metav1.Object
	schema.ObjectKind
}
