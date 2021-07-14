package client

import (
	"github.com/stackrox/rox/operator/pkg/central/clientset/central/typed/central/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	_ kubernetes.Interface              = (*StackRoxV1Alpha1Client)(nil)
	_ v1alpha1.CentralV1alpha1Interface = (*StackRoxV1Alpha1Client)(nil)
)

// StackRoxV1Alpha1Client is a client which is a typed client for its custom resources.
type StackRoxV1Alpha1Client struct {
	*kubernetes.Clientset
	centralClient *v1alpha1.CentralV1alpha1Client
}

// NewForConfigOrDie creates a new kubernetes client
func NewForConfigOrDie(c *rest.Config) *StackRoxV1Alpha1Client {
	clientSet := kubernetes.NewForConfigOrDie(c)
	centralClient := v1alpha1.NewForConfigOrDie(c)

	return &StackRoxV1Alpha1Client{Clientset: clientSet, centralClient: centralClient}
}

// Centrals returns the Central client
func (c *StackRoxV1Alpha1Client) Centrals(namespace string) v1alpha1.CentralInterface {
	return c.centralClient.Centrals(namespace)
}
