package config

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/sensor/upgrader/flags"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func loadKubeConfig(timeout time.Duration) (*rest.Config, error) {
	switch cs := *flags.KubeConfigSource; cs {
	case "in-cluster":
		cfg, err := rest.InClusterConfig()
		if err != nil {
			return nil, errors.Wrap(err, "loading in-cluster Kubernetes config")
		}
		cfg.Timeout = timeout
		return cfg, nil
	case "kubectl":
		return loadKubeCtlConfig(timeout)
	default:
		return nil, errors.Errorf("invalid kube config source %q", cs)
	}
}

func loadKubeCtlConfig(timeout time.Duration) (*rest.Config, error) {
	config, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		return nil, errors.Wrap(err, "loading default Kubernetes client config")
	}

	cfg, err := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{
		Timeout: timeout.String(),
	}).ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "creating Kubernetes client config from kubeconfig")
	}
	return cfg, nil
}
