package aws

import (
	"github.com/stackrox/rox/pkg/cloudproviders/aws"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"k8s.io/client-go/kubernetes"
)

var (
	once    sync.Once
	manager aws.CredentialsManager

	log = logging.LoggerForModule()
)

// Singleton returns an instance of the AWS cloud credentials manager.
func Singleton() aws.CredentialsManager {
	once.Do(func() {
		restCfg, err := k8sutil.GetK8sInClusterConfig()
		if err != nil {
			log.Error("Could create AWS credentials manager. Continuing with default credentials chain: ", err)
			manager = &aws.DefaultCredentialsManager{}
			return
		}
		k8sClient, err := kubernetes.NewForConfig(restCfg)
		if err != nil {
			log.Error("Could create AWS credentials manager. Continuing with default credentials chain: ", err)
			manager = &aws.DefaultCredentialsManager{}
			return
		}
		manager = aws.NewCredentialsManager(
			k8sClient,
			env.Namespace.Setting(),
			env.AWSCloudCredentialsSecret.Setting(),
			"mirrored-awsCloudCredentials",
		)
	})
	return manager
}
