package auth

import (
	"github.com/stackrox/rox/pkg/auth/tokensource"
	"github.com/stackrox/rox/pkg/k8sutil"
	"golang.org/x/oauth2"
	"k8s.io/client-go/kubernetes"
)

type stsTokenManagerImpl struct {
	credManager CredentialsManager
	tokenSource *tokensource.ReuseTokenSourceWithInvalidate
}

var _ STSTokenManager = &stsTokenManagerImpl{}

func fallbackSTSClientManager() STSTokenManager {
	credManager := &defaultCredentialsManager{}
	mgr := &stsTokenManagerImpl{
		credManager: credManager,
		tokenSource: tokensource.NewReuseTokenSourceWithInvalidate(&CredentialManagerTokenSource{credManager}),
	}
	return mgr
}

// NewSTSTokenManager creates a new GCP token manager.
func NewSTSTokenManager(namespace string, secretName string) STSTokenManager {
	restCfg, err := k8sutil.GetK8sInClusterConfig()
	if err != nil {
		log.Error("Could not create GCP credentials manager. Continuing with default credentials chain: ", err)
		return fallbackSTSClientManager()
	}
	k8sClient, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		log.Error("Could not create GCP credentials manager. Continuing with default credentials chain: ", err)
		return fallbackSTSClientManager()
	}
	mgr := &stsTokenManagerImpl{}
	mgr.credManager = newCredentialsManagerImpl(k8sClient, namespace, secretName, mgr.invalidateToken)
	mgr.tokenSource = tokensource.NewReuseTokenSourceWithInvalidate(&CredentialManagerTokenSource{mgr.credManager})
	return mgr
}

func (c *stsTokenManagerImpl) Start() {
	c.credManager.Start()
}

func (c *stsTokenManagerImpl) Stop() {
	c.credManager.Stop()
}

func (c *stsTokenManagerImpl) TokenSource() oauth2.TokenSource {
	return c.tokenSource
}

func (c *stsTokenManagerImpl) invalidateToken() {
	c.tokenSource.Invalidate()
}
