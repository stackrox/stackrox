package migratetooperator

import (
	"context"
	"time"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	// Activate auth providers for client-go.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type clusterSource struct {
	client    kubernetes.Interface
	namespace string
}

func newClusterSource(namespace string) (*clusterSource, error) {
	config, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		return nil, errors.Wrap(err, "loading kubeconfig")
	}

	restConfig, err := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "building REST config from kubeconfig")
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "creating Kubernetes client")
	}

	return &clusterSource{client: client, namespace: namespace}, nil
}

func (s *clusterSource) CentralDeployment() (*appsv1.Deployment, error) {
	return s.getDeployment("central")
}

func (s *clusterSource) CentralDBDeployment() (*appsv1.Deployment, error) {
	return s.getDeployment("central-db")
}

func (s *clusterSource) getDeployment(name string) (*appsv1.Deployment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dep, err := s.client.AppsV1().Deployments(s.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "getting %s Deployment in namespace %q", name, s.namespace)
	}
	return dep, nil
}
