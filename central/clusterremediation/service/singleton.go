package service

import (
	imageIntegration "github.com/stackrox/rox/central/imageintegration"
	"github.com/stackrox/rox/central/operatorremediation/datasource"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"k8s.io/client-go/dynamic"
)

var (
	once sync.Once
	svc  Service

	log = logging.LoggerForModule()
)

// inClusterDynamicClient builds a dynamic Kubernetes client from the in-cluster config, or nil when
// one cannot be built (the service then reports cluster access as unavailable at call time).
func inClusterDynamicClient() dynamic.Interface {
	cfg, err := k8sutil.GetK8sInClusterConfig()
	if err != nil {
		log.Warnf("cluster remediation: no in-cluster Kubernetes config, OpenShift release access disabled: %v", err)
		return nil
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		log.Warnf("cluster remediation: could not build dynamic Kubernetes client: %v", err)
		return nil
	}
	return dyn
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(func() {
		svc = newService(datasource.Singleton(), inClusterDynamicClient(), imageIntegration.Set().RegistrySet())
	})
	return svc
}
