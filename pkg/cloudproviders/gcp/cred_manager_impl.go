package gcp

import (
	"context"

	"github.com/stackrox/rox/pkg/secretinformer"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/oauth2/google"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	cloudCredentialsKey = "credentials"
)

type gcpCredentialsManagerImpl struct {
	informer  *secretinformer.SecretInformer
	stsConfig []byte
	mutex     sync.RWMutex
}

var _ CredentialsManager = &gcpCredentialsManagerImpl{}

// NewCredentialsManager creates a new GCP credential manager.
func NewCredentialsManager(k8sClient kubernetes.Interface, namespace string, secretName string) *gcpCredentialsManagerImpl {
	mgr := &gcpCredentialsManagerImpl{}
	mgr.informer = secretinformer.NewSecretInformer(
		namespace,
		secretName,
		k8sClient,
		mgr.updateSecret,
		mgr.updateSecret,
		mgr.deleteSecret,
	)
	return mgr
}

func (c *gcpCredentialsManagerImpl) updateSecret(secret *v1.Secret) {
	if stsConfig, ok := secret.Data[cloudCredentialsKey]; ok {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		c.stsConfig = []byte(stsConfig)
		log.Infof("Updated GCP cloud credentials based on %s/%s", c.informer.Namespace, c.informer.SecretName)
	}
}

func (c *gcpCredentialsManagerImpl) deleteSecret() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.stsConfig = []byte{}
	log.Infof("Deleted GCP cloud credentials based on %s/%s", c.informer.Namespace, c.informer.SecretName)
}

func (c *gcpCredentialsManagerImpl) Start() error {
	return c.informer.Start()
}

func (c *gcpCredentialsManagerImpl) Stop() {
	c.informer.Stop()
}

// GetCredentials returns GCP credentials based on the environment.
//
// The following sources are considered:
//  1. Cloud credentials secret (stackrox/gcp-cloud-credentials) containing the STS configuration
//     for federated workload identities. Ignored if the secret does not exist.
//  2. The default GCP credentials chain based on the pod's environment and metadata.
func (c *gcpCredentialsManagerImpl) GetCredentials(ctx context.Context) (*google.Credentials, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if len(c.stsConfig) > 0 {
		return google.CredentialsFromJSONWithParams(ctx, c.stsConfig, google.CredentialsParams{})
	}
	return google.FindDefaultCredentials(ctx)
}
