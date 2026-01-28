package k8sutil

import (
	"net"
	"os"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/stringutils"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	clientContentType = stringutils.FirstNonEmpty(env.KubernetesClientContentType.Setting(), "application/vnd.kubernetes.protobuf")
	log               = logging.LoggerForModule()
)

// GetK8sInClusterConfig returns a k8s client config that can be used from within cluster.
// It is adjusted to use DNS record instead of raw IP as API host in certain cases.
// This can be used to more conveniently bypass the proxy.
func GetK8sInClusterConfig() (*rest.Config, error) {
	restCfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	restCfg.ContentType = clientContentType

	// Replacing raw IP address with kubernetes.default.svc
	// allows for easier proxy configuration.
	if env.ManagedCentral.BooleanSetting() {
		port := os.Getenv("KUBERNETES_SERVICE_PORT")
		restCfg.Host = "https://" + net.JoinHostPort("kubernetes.default.svc.cluster.local.", port)
	}
	return restCfg, nil
}

// MustCreateK8sClient creates a k8s client or panics.
func MustCreateK8sClient(config *rest.Config) kubernetes.Interface {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Panicf("Creating Kubernetes clientset: %v", err)
	}
	return client
}
