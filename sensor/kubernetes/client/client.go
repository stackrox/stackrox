package client

import (
	appVersioned "github.com/openshift/client-go/apps/clientset/versioned"
	configVersioned "github.com/openshift/client-go/config/clientset/versioned"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	log = logging.LoggerForModule()
)

// Interface implements an interface that bridges Kubernetes and Openshift
type Interface interface {
	Kubernetes() kubernetes.Interface
	Dynamic() dynamic.Interface
	OpenshiftApps() appVersioned.Interface
	OpenshiftConfig() configVersioned.Interface
}

type clientSet struct {
	dynamic         dynamic.Interface
	k8s             kubernetes.Interface
	openshiftApps   appVersioned.Interface
	openshiftConfig configVersioned.Interface
}

// MustCreateInterface creates a client interface for both Kubernetes and Openshift clients
func MustCreateInterface() Interface {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Panicf("Obtaining in-cluster Kubernetes config: %v", err)
	}

	k8sClientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Panicf("Creating Kubernetes clientset: %v", err)
	}

	var dynamicClientGenerator dynamic.Interface
	if features.ComplianceOperatorCheckResults.Enabled() {
		dynamicClientGenerator, err = dynamic.NewForConfig(config)
		if err != nil {
			log.Panicf("Creating dynamic client: %v", err)
		}
	}

	var openshiftAppsClientSet appVersioned.Interface
	var openshiftConfigClientSet configVersioned.Interface
	if env.OpenshiftAPI.Setting() == "true" {
		config, err := rest.InClusterConfig()
		if err != nil {
			log.Panicf("Unable to get cluster config: %v", err)
		}
		openshiftAppsClientSet, err = appVersioned.NewForConfig(config)
		if err != nil {
			log.Warnf("Could not generate openshift client: %v", err)
		}

		openshiftConfigClientSet, err = configVersioned.NewForConfig(config)
		if err != nil {
			log.Warnf("Could not generate openshift client: %s", err)
		}
	}

	return &clientSet{
		dynamic:         dynamicClientGenerator,
		k8s:             k8sClientSet,
		openshiftApps:   openshiftAppsClientSet,
		openshiftConfig: openshiftConfigClientSet,
	}
}

func (c *clientSet) Kubernetes() kubernetes.Interface {
	return c.k8s
}

func (c *clientSet) OpenshiftApps() appVersioned.Interface {
	return c.openshiftApps
}

func (c *clientSet) OpenshiftConfig() configVersioned.Interface {
	return c.openshiftConfig
}

func (c *clientSet) Dynamic() dynamic.Interface {
	return c.dynamic
}
