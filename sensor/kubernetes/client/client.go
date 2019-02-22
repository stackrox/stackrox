package client

import (
	"github.com/stackrox/rox/pkg/logging"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	log = logging.LoggerForModule()
)

// MustCreateClientSet returns a new Kubernetes clientset, or panics if it can't create one.
func MustCreateClientSet() *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Panicf("Obtaining in-cluster Kubernetes config: %v", err)
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Panicf("Creating Kubernetes clientset: %v", err)
	}

	return clientSet
}
