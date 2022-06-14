package certdistribution

import (
	"github.com/stackrox/stackrox/pkg/grpc"
	"k8s.io/client-go/kubernetes"
)

// NewService creates a new service for certificate distribution.
func NewService(k8sClient kubernetes.Interface, namespace string) grpc.APIService {
	return newService(k8sClient, namespace)
}
