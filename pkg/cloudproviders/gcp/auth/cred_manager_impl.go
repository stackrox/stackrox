package auth

import (
	"bytes"
	"context"

	artifactv1 "cloud.google.com/go/artifactregistry/apiv1"
	securitycenterv1 "cloud.google.com/go/securitycenter/apiv1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/secretinformer"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/oauth2/google"
	storagev1 "google.golang.org/api/storage/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	cloudCredentialsKey = "credentials"
)

var log = logging.LoggerForModule()

type gcpCredentialsManagerImpl struct {
	namespace  string
	secretName string
	onChangeFn func()
	informer   *secretinformer.SecretInformer
	stsConfig  []byte
	mutex      sync.RWMutex
}

var _ CredentialsManager = &gcpCredentialsManagerImpl{}

func newCredentialsManagerImpl(
	k8sClient kubernetes.Interface,
	namespace string,
	secretName string,
	onChangeFn func(),
) *gcpCredentialsManagerImpl {
	mgr := &gcpCredentialsManagerImpl{
		namespace:  namespace,
		secretName: secretName,
		onChangeFn: onChangeFn,
		stsConfig:  []byte{},
	}
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
		var hasChanged bool
		defer func() {
			if hasChanged {
				c.onChangeFn()
			}
		}()

		c.mutex.Lock()
		defer c.mutex.Unlock()
		if bytes.Equal(c.stsConfig, stsConfig) {
			return
		}

		hasChanged = true
		c.stsConfig = stsConfig
		log.Infof("Updated GCP cloud credentials based on %s/%s", c.namespace, c.secretName)
	}
}

func (c *gcpCredentialsManagerImpl) deleteSecret() {
	defer c.onChangeFn()
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.stsConfig = []byte{}
	log.Infof("Deleted GCP cloud credentials based on %s/%s", c.namespace, c.secretName)
}

func (c *gcpCredentialsManagerImpl) Start() {
	if err := c.informer.Start(); err != nil {
		log.Error("Failed to start GCP cloud credentials manager: ", err)
	}
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
	scopes := []string{storagev1.CloudPlatformScope}
	scopes = append(scopes, artifactv1.DefaultAuthScopes()...)
	scopes = append(scopes, securitycenterv1.DefaultAuthScopes()...)

	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if len(c.stsConfig) > 0 {
		// Use a scope to request access to the GCP API. See
		// https://developers.google.com/identity/protocols/oauth2/scopes
		// for a list of GCP scopes.
		return google.CredentialsFromJSON(ctx, c.stsConfig, scopes...)
	}
	return google.FindDefaultCredentials(ctx, scopes...)
}
