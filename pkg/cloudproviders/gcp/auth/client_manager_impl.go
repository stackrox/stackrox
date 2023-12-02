package auth

import (
	"context"
	"time"

	securitycenter "cloud.google.com/go/securitycenter/apiv1"
	"cloud.google.com/go/storage"
	"github.com/stackrox/rox/pkg/cloudproviders/gcp/handler"
	"github.com/stackrox/rox/pkg/k8sutil"
	"k8s.io/client-go/kubernetes"
)

const updateTimeout = 1 * time.Hour

type stsClientManagerImpl struct {
	credManager                 CredentialsManager
	storageClientHandler        handler.Handler[*storage.Client]
	securityCenterClientHandler handler.Handler[*securitycenter.Client]
}

var _ STSClientManager = &stsClientManagerImpl{}

func fallbackSTSClientManager() STSClientManager {
	mgr := &stsClientManagerImpl{
		credManager:                 &defaultCredentialsManager{},
		storageClientHandler:        handler.NewHandlerNoInit[*storage.Client](),
		securityCenterClientHandler: handler.NewHandlerNoInit[*securitycenter.Client](),
	}
	mgr.updateClients()
	return mgr
}

// NewSTSClientManager creates a new GCP client manager.
func NewSTSClientManager(namespace string, secretName string) STSClientManager {
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
	mgr := &stsClientManagerImpl{storageClientHandler: handler.NewHandlerNoInit[*storage.Client]()}
	mgr.credManager = newCredentialsManagerImpl(k8sClient, namespace, secretName, mgr.updateClients)
	mgr.updateClients()
	return mgr
}

func (c *stsClientManagerImpl) Start() {
	c.credManager.Start()
}

func (c *stsClientManagerImpl) Stop() {
	c.credManager.Stop()
}

func (c *stsClientManagerImpl) updateClients() {
	ctx, cancel := context.WithTimeout(context.Background(), updateTimeout)
	defer cancel()
	creds, err := c.credManager.GetCredentials(ctx)
	if err != nil {
		log.Error("Failed to get GCP credentials: ", err)
		return
	}

	if err := c.storageClientHandler.UpdateClient(ctx, creds); err != nil {
		log.Error("Failed to update GCP storage client: ", err)
	}
}

func (c *stsClientManagerImpl) StorageClientHandler() handler.Handler[*storage.Client] {
	return c.storageClientHandler
}

func (c *stsClientManagerImpl) SecurityCenterClientHandler() handler.Handler[*securitycenter.Client] {
	return c.securityCenterClientHandler
}
