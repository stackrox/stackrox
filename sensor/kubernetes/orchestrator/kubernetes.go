package orchestrator

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/pkg/env"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/orchestrators"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	logger       = logging.New("orchestrator")
	deleteOption = &[]metav1.DeletionPropagation{metav1.DeletePropagationForeground}[0]
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

func (k *kubernetesOrchestrator) Launch(service orchestrators.SystemService) (string, error) {
	if service.Global {
		ds := k.converter.asDaemonSet(k.newServiceWrap(service))
		if _, err := k.client.ExtensionsV1beta1().DaemonSets(k.namespace).Create(ds); err != nil {
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

func (k *kubernetesOrchestrator) newServiceWrap(service orchestrators.SystemService) *serviceWrap {
	return &serviceWrap{
		SystemService: service,
		namespace:     k.namespace,
	}
}

func (k *kubernetesOrchestrator) Kill(name string) error {
	if ds, err := k.client.ExtensionsV1beta1().DaemonSets(k.namespace).Get(name, metav1.GetOptions{}); err == nil && ds != nil {
		if err := k.client.ExtensionsV1beta1().DaemonSets(k.namespace).Delete(name, &metav1.DeleteOptions{PropagationPolicy: deleteOption}); err != nil {
			logger.Errorf("unable to delete daemonset %s: %s", name, err)
			return err
		}
		return nil
	}

	if deploy, err := k.client.ExtensionsV1beta1().Deployments(k.namespace).Get(name, metav1.GetOptions{}); err == nil && deploy != nil {
		if err := k.client.ExtensionsV1beta1().Deployments(k.namespace).Delete(name, &metav1.DeleteOptions{PropagationPolicy: deleteOption}); err != nil {
			logger.Errorf("unable to delete deployment %s: %s", name, err)
			return err
		}
		return nil
	}

	err := fmt.Errorf("unable to delete service %s; service not found", name)
	logger.Error(err)
	return err
}
