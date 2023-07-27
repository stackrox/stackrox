package client

import (
	appVersioned "github.com/openshift/client-go/apps/clientset/versioned"
	configVersioned "github.com/openshift/client-go/config/clientset/versioned"
	operatorVersioned "github.com/openshift/client-go/operator/clientset/versioned"
	routeVersioned "github.com/openshift/client-go/route/clientset/versioned"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/stringutils"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	clientContentType = stringutils.FirstNonEmpty(env.KubernetesClientContentType.Setting(), "application/vnd.kubernetes.protobuf")

	log = logging.LoggerForModule()
)

// Interface implements an interface that bridges Kubernetes and Openshift
type Interface interface {
	Kubernetes() kubernetes.Interface
	Dynamic() dynamic.Interface
	OpenshiftApps() appVersioned.Interface
	OpenshiftConfig() configVersioned.Interface
	OpenshiftRoute() routeVersioned.Interface
	OpenshiftOperator() operatorVersioned.Interface
}

type clientSet struct {
	dynamic           dynamic.Interface
	k8s               kubernetes.Interface
	openshiftApps     appVersioned.Interface
	openshiftConfig   configVersioned.Interface
	openshiftRoute    routeVersioned.Interface
	openshiftOperator operatorVersioned.Interface
}

func mustCreateK8sClient(config *rest.Config) kubernetes.Interface {
	config.ContentType = clientContentType
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Panicf("Creating Kubernetes clientset: %v", err)
	}
	return client
}

func mustCreateOpenshiftRouteClient(config *rest.Config) routeVersioned.Interface {
	if !env.OpenshiftAPI.BooleanSetting() {
		return nil
	}
	client, err := routeVersioned.NewForConfig(config)
	if err != nil {
		log.Panicf("Could not generate openshift routes client: %v", err)
	}
	return client
}

func mustCreateOpenshiftAppsClient(config *rest.Config) appVersioned.Interface {
	if !env.OpenshiftAPI.BooleanSetting() {
		return nil
	}
	client, err := appVersioned.NewForConfig(config)
	if err != nil {
		log.Panicf("Could not generate openshift apps client: %v", err)
	}
	return client
}

func mustCreateOpenshiftConfigClient(config *rest.Config) configVersioned.Interface {
	if !env.OpenshiftAPI.BooleanSetting() {
		return nil
	}
	client, err := configVersioned.NewForConfig(config)
	if err != nil {
		log.Warnf("Could not generate openshift config client: %s", err)
	}
	return client
}

func mustCreateOpenshiftOperatorClient(config *rest.Config) operatorVersioned.Interface {
	if !env.OpenshiftAPI.BooleanSetting() {
		return nil
	}
	client, err := operatorVersioned.NewForConfig(config)
	if err != nil {
		log.Warnf("Could not generate openshift operator client: %s", err)
	}
	return client
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
	config.ContentType = clientContentType
	return &clientSet{
		dynamic:           mustCreateDynamicClient(config),
		k8s:               mustCreateK8sClient(config),
		openshiftApps:     mustCreateOpenshiftAppsClient(config),
		openshiftConfig:   mustCreateOpenshiftConfigClient(config),
		openshiftRoute:    mustCreateOpenshiftRouteClient(config),
		openshiftOperator: mustCreateOpenshiftOperatorClient(config),
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

func (c *clientSet) Kubernetes() kubernetes.Interface {
	return c.k8s
}

func (c *clientSet) OpenshiftApps() appVersioned.Interface {
	return c.openshiftApps
}

func (c *clientSet) OpenshiftConfig() configVersioned.Interface {
	return c.openshiftConfig
}

func (c *clientSet) OpenshiftRoute() routeVersioned.Interface {
	return c.openshiftRoute
}

func (c *clientSet) OpenshiftOperator() operatorVersioned.Interface {
	return c.openshiftOperator
}

func (c *clientSet) Dynamic() dynamic.Interface {
	return c.dynamic
}
