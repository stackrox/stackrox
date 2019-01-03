package listener

import (
	openshift "github.com/openshift/client-go/apps/clientset/versioned"
	"github.com/stackrox/rox/pkg/env"
	kubernetesClient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type clientSet struct {
	k8s       *kubernetesClient.Clientset
	openshift *openshift.Clientset
}

func createClient() *clientSet {
	config, err := rest.InClusterConfig()
	if err != nil {
		logger.Fatalf("Unable to get cluster config: %s", err)
	}

	cs := &clientSet{}
	cs.k8s, err = kubernetesClient.NewForConfig(config)
	if err != nil {
		logger.Fatalf("Unable to get k8s client: %s", err)
	}
	if env.OpenshiftAPI.Setting() == "true" {
		cs.openshift, err = openshift.NewForConfig(config)
		if err != nil {
			logger.Warnf("Could not generate openshift client: %s", err)
		}
	}
	return cs
}
