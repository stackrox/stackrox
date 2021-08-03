package client

import (
	clientset "github.com/stackrox/rox/operator/pkg/clientset/stackrox"
	typedClientset "github.com/stackrox/rox/operator/pkg/clientset/stackrox/typed/platform/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	_ kubernetes.Interface = (*stackRoxClientset)(nil)
	_ StackRoxInterface    = (*stackRoxClientset)(nil)
)

// StackRoxInterface is the interface for the StackRox client
type StackRoxInterface interface {
	kubernetes.Interface
	SecuredClusterV1Alpha1(namespace string) typedClientset.SecuredClusterInterface
	CentralV1Alpha1(namespace string) typedClientset.CentralInterface
}

type stackRoxClientset struct {
	*kubernetes.Clientset
	stackroxClientset clientset.Interface
}

// SecuredClusterV1Alpha1 returns a client to access SecuredCluster resources
func (s stackRoxClientset) SecuredClusterV1Alpha1(namespace string) typedClientset.SecuredClusterInterface {
	return s.stackroxClientset.PlatformV1alpha1().SecuredClusters(namespace)
}

//CentralV1Alpha1 returns a client to access Central resources
func (s stackRoxClientset) CentralV1Alpha1(namespace string) typedClientset.CentralInterface {
	return s.stackroxClientset.PlatformV1alpha1().Centrals(namespace)
}

// NewForConfigOrDie creates a new kubernetes client
func NewForConfigOrDie(c *rest.Config) StackRoxInterface {
	return &stackRoxClientset{
		stackroxClientset: clientset.NewForConfigOrDie(c),
		Clientset:         kubernetes.NewForConfigOrDie(c),
	}
}
