package config

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/sensor/upgrader/flags"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func loadKubeConfig() (*rest.Config, error) {
	switch cs := *flags.KubeConfigSource; cs {
	case "in-cluster":
		return rest.InClusterConfig()
	case "kubectl":
		return loadKubeCtlConfig()
	default:
		return nil, errors.Errorf("invalid kube config source %q", cs)
	}
}

func loadKubeCtlConfig() (*rest.Config, error) {
	config, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		return nil, errors.Wrap(err, "loading default Kubernetes client config")
	}

	return clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
}
