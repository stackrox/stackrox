package k8sutil

import (
	"net"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// GetK8sInClusterConfig returns a k8s client config that can be used from within cluster.
// It is adjusted to use DNS record instead of raw IP as API host in certain cases.
// This can be used to more conveniently bypass the proxy.
func GetK8sInClusterConfig() (*rest.Config, error) {
	restCfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	// Replacing raw IP address with kubernetes.default.svc
	// allows for easier proxy configuration.
	if env.ManagedCentral.BooleanSetting() {
		port := os.Getenv("KUBERNETES_SERVICE_PORT")
		restCfg.Host = "https://" + net.JoinHostPort("kubernetes.default.svc.cluster.local.", port)
	}
	return restCfg, nil
}

// GetK8sInClusterClient returns a k8s client that can be used from within cluster.
func GetK8sInClusterClient() (*kubernetes.Clientset, error) {
	restCfg, err := GetK8sInClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "loading K8s client config")
	}
	return kubernetes.NewForConfig(restCfg)
}
