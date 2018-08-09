package orchestrator

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/benchmarks"
	"github.com/stackrox/rox/pkg/env"
	pkgKubernetes "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/orchestrators"
	v1beta12 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
	"k8s.io/client-go/rest"
)

var (
	logger = logging.LoggerForModule()
)

type kubernetesOrchestrator struct {
	client    *kubernetes.Clientset
	converter converter
	namespace string
}

// New returns a new kubernetes orchestrator client.
func New() (orchestrators.Orchestrator, error) {
	c, err := setupClient()
	if err != nil {
		logger.Errorf("unable to create kubernetes client: %s", err)
		return nil, err
	}
	return &kubernetesOrchestrator{
		client:    c,
		converter: newConverter(),
		namespace: env.Namespace.Setting(),
	}, nil
}

func setupClient() (client *kubernetes.Clientset, err error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return
	}

	return kubernetes.NewForConfig(config)
}

func (k *kubernetesOrchestrator) launch(setInterface v1beta1.DaemonSetInterface, ds *v1beta12.DaemonSet) error {
	for i := 0; i < 3; i++ {
		if _, err := k.client.ExtensionsV1beta1().DaemonSets(k.namespace).Create(ds); err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && statusErr.Status().Reason == metav1.StatusReasonAlreadyExists {
				time.Sleep(10 * time.Second)
				continue
			}
			return err
		}
		return nil
	}
	return nil
}

func (k *kubernetesOrchestrator) Launch(service orchestrators.SystemService) (string, error) {
	if service.Global {
		ds := k.converter.asDaemonSet(k.newServiceWrap(service))
		if err := k.launch(k.client.ExtensionsV1beta1().DaemonSets(k.namespace), ds); err != nil {
			logger.Errorf("unable to create daemonset %s: %s", service.Name, err)
			return "", err
		}
		return service.Name, nil
	}

	deploy := k.converter.asDeployment(k.newServiceWrap(service))
	if _, err := k.client.ExtensionsV1beta1().Deployments(k.namespace).Create(deploy); err != nil {
		logger.Errorf("unable to create deployment %s: %s", service.Name, err)
		return "", err
	}

	return service.Name, nil
}

func (k *kubernetesOrchestrator) LaunchBenchmark(service orchestrators.SystemService) (string, error) {
	service.Command = []string{benchmarks.BenchmarkCommand}
	service.Envs = append(service.Envs, env.Combine(env.BenchmarkCompletion.EnvVar(), "true"))
	service.Mounts = benchmarks.BenchmarkMounts
	service.HostPID = true
	return k.Launch(service)
}

func (k *kubernetesOrchestrator) newServiceWrap(service orchestrators.SystemService) *serviceWrap {
	return &serviceWrap{
		SystemService: service,
		namespace:     k.namespace,
	}
}

func (k *kubernetesOrchestrator) Kill(name string) error {
	if ds, err := k.client.ExtensionsV1beta1().DaemonSets(k.namespace).Get(name, metav1.GetOptions{}); err == nil && ds != nil {
		if err := k.client.ExtensionsV1beta1().DaemonSets(k.namespace).Delete(name, pkgKubernetes.DeleteOption); err != nil {
			logger.Errorf("unable to delete daemonset %s: %s", name, err)
			return err
		}
		return nil
	}

	if deploy, err := k.client.ExtensionsV1beta1().Deployments(k.namespace).Get(name, metav1.GetOptions{}); err == nil && deploy != nil {
		if err := k.client.ExtensionsV1beta1().Deployments(k.namespace).Delete(name, pkgKubernetes.DeleteOption); err != nil {
			logger.Errorf("unable to delete deployment %s: %s", name, err)
			return err
		}
		return nil
	}

	err := fmt.Errorf("unable to delete service %s; service not found", name)
	logger.Error(err)
	return err
}

// WaitForCompletion currently cannot be implemented in Kubernetes because DaemonSet Restart Policy must be always
func (k *kubernetesOrchestrator) WaitForCompletion(_ string, timeout time.Duration) error {
	time.Sleep(timeout)
	return nil
}
