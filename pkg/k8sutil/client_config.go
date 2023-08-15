package k8sutil

import (
	"net"
	"os"

	"github.com/stackrox/rox/pkg/env"
	"k8s.io/client-go/rest"
)

// GetK8sInClusterConfig returns k8s client config that can be used from within cluster.
// It is adjusted to use DNS record instead of raw IP as API host in certain cases.
func GetK8sInClusterConfig() (*rest.Config, error) {
	restCfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	// Replacing raw IP address with kubernetes.default.svc
	// allows for easier proxy configuration.
	if env.ManagedCentral.BooleanSetting() {
		port := os.Getenv("KUBERNETES_SERVICE_PORT")
		restCfg.Host = "https://" + net.JoinHostPort("kubernetes.default.svc", port)
	}
	return restCfg, nil
}
