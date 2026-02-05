package remote

import (
	"context"
	"fmt"
	"os"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	log = logging.LoggerForModule()
)

const (
	// kubeconfigKey is the expected key name in the secret containing the kubeconfig
	kubeconfigKey = "kubeconfig"
)

// RemoteClientManager encapsulates connecting to a remote Kubernetes cluster using configuration from a secret
type RemoteClientManager struct {
	secretName      string
	secretNamespace string
	remoteClient    client.Interface
}

// NewRemoteClientManager creates a new remote client manager if the remote cluster feature is enabled
// Returns nil if ROX_REMOTE_CLUSTER_SECRET is not set
func NewRemoteClientManager() *RemoteClientManager {
	secretName := env.RemoteClusterSecretName.Setting()
	if secretName == "" {
		return nil
	}

	secretNamespace := env.RemoteClusterSecretNamespace.Setting()
	if secretNamespace == "" {
		// Default to the current pod's namespace
		namespace, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			log.Warnf("Failed to read pod namespace, defaulting to 'stackrox': %v", err)
			secretNamespace = "stackrox"
		} else {
			secretNamespace = string(namespace)
		}
	}

	mgr := &RemoteClientManager{
		secretName:      secretName,
		secretNamespace: secretNamespace,
	}

	log.Infof("Remote cluster mode enabled: will read kubeconfig from secret %s/%s", secretNamespace, secretName)
	return mgr
}

// InitializeRemoteClient reads the kubeconfig from the specified secret and creates a client
// This requires access to the local cluster to read the secret, so we need a local client
func (r *RemoteClientManager) InitializeRemoteClient(localClient kubernetes.Interface) error {
	ctx := context.Background()

	log.Infof("Reading kubeconfig from secret %s/%s", r.secretNamespace, r.secretName)

	// Read the secret containing the kubeconfig
	secret, err := localClient.CoreV1().Secrets(r.secretNamespace).Get(ctx, r.secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to read remote cluster secret %s/%s: %w", r.secretNamespace, r.secretName, err)
	}

	// Extract kubeconfig from secret
	kubeconfigData, ok := secret.Data[kubeconfigKey]
	if !ok {
		return fmt.Errorf("secret %s/%s does not contain key '%s'", r.secretNamespace, r.secretName, kubeconfigKey)
	}

	log.Infof("Successfully read kubeconfig from secret (size: %d bytes)", len(kubeconfigData))

	// Parse kubeconfig
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigData)
	if err != nil {
		return fmt.Errorf("failed to parse kubeconfig from secret: %w", err)
	}

	// Create remote client
	r.remoteClient = client.MustCreateInterfaceFromRest(restConfig)

	log.Infof("Successfully created remote cluster client for server: %s", restConfig.Host)
	return nil
}

// Client returns the remote Kubernetes client interface
func (r *RemoteClientManager) Client() client.Interface {
	return r.remoteClient
}
