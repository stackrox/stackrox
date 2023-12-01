package auth

import (
	"context"

	"github.com/stackrox/rox/pkg/cloudproviders/gcp/storage"
	"github.com/stackrox/rox/pkg/k8sutil"
	"k8s.io/client-go/kubernetes"
)

type stsClientManagerImpl struct {
	credManager          CredentialsManager
	storageClientHandler storage.ClientHandler
}

var _ STSClientManager = &stsClientManagerImpl{}

func fallbackSTSClientManager() STSClientManager {
	mgr := &stsClientManagerImpl{
		credManager:          &defaultCredentialsManager{},
		storageClientHandler: storage.NewClientHandlerNoInit(),
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
	mgr := &stsClientManagerImpl{storageClientHandler: storage.NewClientHandlerNoInit()}
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
	ctx := context.Background()
	creds, err := c.credManager.GetCredentials(ctx)
	if err != nil {
		log.Error("Failed to get GCP credentials: ", err)
		return
	}

	if err := c.storageClientHandler.UpdateClient(ctx, creds); err != nil {
		log.Error("Failed to update GCP storage client: ", err)
	}
}

func (c *stsClientManagerImpl) StorageClientHandler() storage.ClientHandler {
	return c.storageClientHandler
}
