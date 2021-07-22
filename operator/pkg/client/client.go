package client

import (
	central "github.com/stackrox/rox/operator/pkg/clientset/stackrox"
	securedcluster "github.com/stackrox/rox/operator/pkg/clientset/stackrox"
	centralv1alpha1 "github.com/stackrox/rox/operator/pkg/clientset/stackrox/typed/platform/v1alpha1"
	securedclusterv1alpha1 "github.com/stackrox/rox/operator/pkg/clientset/stackrox/typed/platform/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	_ kubernetes.Interface = (*StackRoxClientset)(nil)
	_ StackRoxInterface    = (*StackRoxClientset)(nil)
)

// StackRoxInterface is the interface for the StackRox client
type StackRoxInterface interface {
	kubernetes.Interface
	SecuredClusterV1Alpha1(namespace string) securedclusterv1alpha1.SecuredClusterInterface
	CentralV1Alpha1(namespace string) centralv1alpha1.CentralInterface
}

// StackRoxClientset is a kubernetes client to access StackRox resources
type StackRoxClientset struct {
	*kubernetes.Clientset
	centralClientSet        central.Interface
	securedClusterClientSet securedcluster.Interface
}

// SecuredClusterV1Alpha1 returns a client to access SecuredCluster resources
func (s StackRoxClientset) SecuredClusterV1Alpha1(namespace string) securedclusterv1alpha1.SecuredClusterInterface {
	return s.securedClusterClientSet.PlatformV1alpha1().SecuredClusters(namespace)
}

//CentralV1Alpha1 returns a client to access Central resources
func (s StackRoxClientset) CentralV1Alpha1(namespace string) centralv1alpha1.CentralInterface {
	return s.centralClientSet.PlatformV1alpha1().Centrals(namespace)
}

// NewForConfigOrDie creates a new kubernetes client
func NewForConfigOrDie(c *rest.Config) StackRoxInterface {
	client := kubernetes.NewForConfigOrDie(c)

	return &StackRoxClientset{
		securedClusterClientSet: securedcluster.NewForConfigOrDie(c),
		centralClientSet:        central.NewForConfigOrDie(c),
		Clientset:               client,
	}
}
