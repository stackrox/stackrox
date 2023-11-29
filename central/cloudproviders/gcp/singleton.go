package gcp

import (
	"github.com/stackrox/rox/pkg/cloudproviders/gcp"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"k8s.io/client-go/kubernetes"
)

var (
	once    sync.Once
	manager gcp.CredentialsManager

	log = logging.LoggerForModule()
)

// Singleton returns an instance of the GCP cloud credentials manager.
func Singleton() gcp.CredentialsManager {
	once.Do(func() {
		restCfg, err := k8sutil.GetK8sInClusterConfig()
		if err != nil {
			log.Error("Could not create GCP credentials manager. Continuing with default credentials chain: ", err)
			manager = &gcp.DefaultCredentialsManager{}
			return
		}
		k8sClient, err := kubernetes.NewForConfig(restCfg)
		if err != nil {
			log.Error("Could not create GCP credentials manager. Continuing with default credentials chain: ", err)
			manager = &gcp.DefaultCredentialsManager{}
			return
		}
		manager = gcp.NewCredentialsManager(
			k8sClient,
			env.Namespace.Setting(),
			env.GCPCloudCredentialsSecret.Setting(),
		)
	})
	return manager
}
