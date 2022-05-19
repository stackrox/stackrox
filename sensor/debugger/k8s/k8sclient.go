package k8s

import (
	appVersioned "github.com/openshift/client-go/apps/clientset/versioned"
	configVersioned "github.com/openshift/client-go/config/clientset/versioned"
	routeVersioned "github.com/openshift/client-go/route/clientset/versioned"
	"github.com/pkg/errors"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8sConfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

// MakeFakeClient creates a k8s client that is not connected to any cluster
func MakeFakeClient() *ClientSet {
	return &ClientSet{
		k8s: fake.NewSimpleClientset(),
	}
}

// ClientSet is a test version of kubernetes.ClientSet
type ClientSet struct {
	dynamic         dynamic.Interface
	k8s             kubernetes.Interface
	openshiftApps   appVersioned.Interface
	openshiftConfig configVersioned.Interface
	openshiftRoute  routeVersioned.Interface
}

// MakeOutOfClusterClient creates a k8s client that uses host configuration to connect to a cluster.
// If host machine has a KUBECONFIG env set it will use it to connect to the respective cluster.
func MakeOutOfClusterClient() (*ClientSet, error) {
	config, err := k8sConfig.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "getting k8s config")
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "creating ClientSet")
	}

	return &ClientSet{
		k8s: k8sClient,
	}, nil
}

// Kubernetes returns the kubernetes interface
func (c *ClientSet) Kubernetes() kubernetes.Interface {
	return c.k8s
}

// OpenshiftApps returns the OpenshiftApps interface
// This is not used in tests!
func (c *ClientSet) OpenshiftApps() appVersioned.Interface {
	return c.openshiftApps
}

// OpenshiftConfig returns the OpenshiftConfig interface
// This is not used in tests!
func (c *ClientSet) OpenshiftConfig() configVersioned.Interface {
	return c.openshiftConfig
}

// OpenshiftRoute returns the OpenshiftRoute interface
// This is not used in tests!
func (c *ClientSet) OpenshiftRoute() routeVersioned.Interface {
	return c.openshiftRoute
}

// Dynamic returns the Dynamic interface
// This is not used in tests!
func (c *ClientSet) Dynamic() dynamic.Interface {
	return c.dynamic
}
