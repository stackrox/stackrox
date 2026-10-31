package service

import (
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

// inClusterDynamicClient builds a dynamic Kubernetes client from the in-cluster config. It returns
// nil (not fatal) when a config cannot be built — the service then reports OLM access as
// unavailable at call time rather than preventing Central from starting.
func inClusterDynamicClient() dynamic.Interface {
	cfg, err := k8sutil.GetK8sInClusterConfig()
	if err != nil {
		log.Warnf("operator remediation: no in-cluster Kubernetes config, OLM access disabled: %v", err)
		return nil
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		log.Warnf("operator remediation: could not build dynamic Kubernetes client, OLM access disabled: %v", err)
		return nil
	}
	return dyn
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(func() {
		svc = newService(datasource.Singleton(), inClusterDynamicClient())
	})
	return svc
}
