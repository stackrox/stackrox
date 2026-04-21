package migratetooperator

import (
	"context"

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

func (s *clusterSource) CentralDBDeployment() (*appsv1.Deployment, error) {
	dep, err := s.client.AppsV1().Deployments(s.namespace).Get(context.Background(), "central-db", metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "getting central-db Deployment in namespace %q", s.namespace)
	}
	return dep, nil
}
