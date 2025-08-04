package certdistribution

import (
	"github.com/stackrox/rox/pkg/grpc"
	"k8s.io/client-go/kubernetes"
)

// NewService creates a new service for certificate distribution.
func NewService(clusterIDGetter clusterIDGetter, k8sClient kubernetes.Interface, namespace string) grpc.APIService {
	return newService(clusterIDGetter, k8sClient, namespace)
}

type clusterIDGetter interface {
	Get() string
}
