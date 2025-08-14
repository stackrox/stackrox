package certdistribution

import (
	"github.com/stackrox/rox/pkg/grpc"
	"k8s.io/client-go/kubernetes"
)

// NewService creates a new service for certificate distribution.
func NewService(clusterID clusterIDWaiter, k8sClient kubernetes.Interface, namespace string) grpc.APIService {
	return newService(clusterID, k8sClient, namespace)
}

type clusterIDWaiter interface {
	Get() string
}
