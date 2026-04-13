package client

import (
	"github.com/stackrox/rox/pkg/logging"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

var (
	log = logging.LoggerForModule()
)

// Interface provides access to the dynamic Kubernetes client and discovery.
// All CRUD operations use the dynamic client with GVR constants from gvr.go,
// eliminating ~113 typed client-go packages (informers, listers, applyconfigurations).
type Interface interface {
	Dynamic() dynamic.Interface
	Discovery() discovery.DiscoveryInterface
}

type clientSet struct {
	dynamic   dynamic.Interface
	discovery discovery.DiscoveryInterface
}

func mustCreateDynamicClient(config *rest.Config) dynamic.Interface {
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Panicf("Creating dynamic client: %v", err)
	}
	return client
}

func mustCreateDiscoveryClient(config *rest.Config) discovery.DiscoveryInterface {
	client, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		log.Panicf("Creating discovery client: %v", err)
	}
	return client
}

// MustCreateInterfaceFromRest creates a client interface using a rest config as a parameter
func MustCreateInterfaceFromRest(config *rest.Config) Interface {
	return &clientSet{
		dynamic:   mustCreateDynamicClient(config),
		discovery: mustCreateDiscoveryClient(config),
	}
}

// MustCreateInterface creates a client interface for both Kubernetes and Openshift clients
func MustCreateInterface() Interface {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Panicf("Obtaining in-cluster Kubernetes config: %v", err)
	}
	return MustCreateInterfaceFromRest(config)
}

func (c *clientSet) Dynamic() dynamic.Interface {
	return c.dynamic
}

func (c *clientSet) Discovery() discovery.DiscoveryInterface {
	return c.discovery
}
