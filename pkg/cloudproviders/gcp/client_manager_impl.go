package gcp

import (
	"context"
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/sync"
	"google.golang.org/api/option"
	"k8s.io/client-go/kubernetes"
)

type stsClientManagerImpl struct {
	credManager   credentialsManager
	storageClient *storage.Client
	mutex         sync.Mutex
	waitGroup     sync.WaitGroup
}

var _ STSClientManager = &stsClientManagerImpl{}

// NewSTSClientManager creates a new GCP client manager.
func NewSTSClientManager(namespace string, secretName string) STSClientManager {
	restCfg, err := k8sutil.GetK8sInClusterConfig()
	if err != nil {
		log.Error("Could not create GCP credentials manager. Continuing with default credentials chain: ", err)
		mgr := &stsClientManagerImpl{credManager: &defaultCredentialsManager{}}
		mgr.updateClients()
		return mgr
	}
	k8sClient, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		log.Error("Could not create GCP credentials manager. Continuing with default credentials chain: ", err)
		mgr := &stsClientManagerImpl{credManager: &defaultCredentialsManager{}}
		mgr.updateClients()
		return mgr
	}
	mgr := &stsClientManagerImpl{}
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
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.waitGroup.Wait()

	creds, err := c.credManager.GetCredentials(ctx)
	if err != nil {
		log.Error("failed to get GCP credentials: ", err)
		return
	}

	c.storageClient, err = storage.NewClient(ctx,
		option.WithCredentials(creds),
		option.WithHTTPClient(&http.Client{Transport: proxy.RoundTripper()}),
	)
	if err != nil {
		log.Error("failed to create GCP storage client: ", err)
	}
}

func (c *stsClientManagerImpl) StorageClient(ctx context.Context) (*storage.Client, func()) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.waitGroup.Add(1)
	return c.storageClient, func() { c.waitGroup.Done() }
}
