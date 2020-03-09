package client

import (
	"github.com/openshift/client-go/apps/clientset/versioned"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	log = logging.LoggerForModule()
)

// Interface implements an interface that bridges Kubernetes and Openshift
type Interface interface {
	Kubernetes() kubernetes.Interface
	Openshift() versioned.Interface
}

type clientSet struct {
	k8s       kubernetes.Interface
	openshift versioned.Interface
}

// MustCreateInterface creates a client interface for both Kubernetes and Openshfit clients
func MustCreateInterface() Interface {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Panicf("Obtaining in-cluster Kubernetes config: %v", err)
	}

	k8sClientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Panicf("Creating Kubernetes clientset: %v", err)
	}

	var openshiftClientSet versioned.Interface
	if env.OpenshiftAPI.Setting() == "true" {
		config, err := rest.InClusterConfig()
		if err != nil {
			log.Fatalf("Unable to get cluster config: %s", err)
		}
		openshiftClientSet, err = versioned.NewForConfig(config)
		if err != nil {
			log.Warnf("Could not generate openshift client: %s", err)
		}
	}
	return &clientSet{
		k8s:       k8sClientSet,
		openshift: openshiftClientSet,
	}
}

func (c *clientSet) Kubernetes() kubernetes.Interface {
	return c.k8s
}

func (c *clientSet) Openshift() versioned.Interface {
	return c.openshift
}
