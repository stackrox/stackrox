package client

import (
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	log = logging.LoggerForModule()
)

// Interface provides access to Kubernetes and dynamic clients.
// OpenShift resources are accessed via the Dynamic() client with GVR constants,
// eliminating the need to import typed OpenShift client-go packages that
// register scheme types at init() (~10 MB RSS overhead).
type Interface interface {
	Kubernetes() kubernetes.Interface
	Dynamic() dynamic.Interface
}

type clientSet struct {
	dynamic dynamic.Interface
	k8s     kubernetes.Interface
}

func mustCreateDynamicClient(config *rest.Config) dynamic.Interface {
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Panicf("Creating dynamic client: %v", err)
	}
	return client
}

// MustCreateInterfaceFromRest creates a client interface using a rest config as a parameter
func MustCreateInterfaceFromRest(config *rest.Config) Interface {
	return &clientSet{
		dynamic: mustCreateDynamicClient(config),
		k8s:     k8sutil.MustCreateK8sClient(config),
	}
}

// MustCreateInterface creates a client interface for both Kubernetes and Openshift clients
func MustCreateInterface() Interface {
	config, err := k8sutil.GetK8sInClusterConfig()
	if err != nil {
		log.Panicf("Obtaining in-cluster Kubernetes config: %v", err)
	}
	return MustCreateInterfaceFromRest(config)
}

func (c *clientSet) Kubernetes() kubernetes.Interface {
	return c.k8s
}

func (c *clientSet) Dynamic() dynamic.Interface {
	return c.dynamic
}
