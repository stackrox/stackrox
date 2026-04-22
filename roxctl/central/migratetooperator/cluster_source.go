package migratetooperator

import (
	"context"
	"time"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	// Activate auth providers for client-go.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type clusterSource struct {
	typed     kubernetes.Interface
	dynamic   dynamic.Interface
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

	typedClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "creating Kubernetes client")
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "creating dynamic Kubernetes client")
	}

	return &clusterSource{typed: typedClient, dynamic: dynamicClient, namespace: namespace}, nil
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

	dep, err := s.typed.AppsV1().Deployments(s.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "getting %s Deployment in namespace %q", name, s.namespace)
	}
	return dep, nil
}

var kindToGVR = map[string]schema.GroupVersionResource{
	"Service": {Group: "", Version: "v1", Resource: "services"},
	"Secret":  {Group: "", Version: "v1", Resource: "secrets"},
	"Route":   {Group: "route.openshift.io", Version: "v1", Resource: "routes"},
}

func (s *clusterSource) ResourceByKindAndName(kind, name string) (bool, map[string]interface{}, error) {
	gvr, ok := kindToGVR[kind]
	if !ok {
		return false, nil, errors.Errorf("unsupported resource kind %q", kind)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	obj, err := s.dynamic.Resource(gvr).Namespace(s.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return false, nil, nil
		}
		return false, nil, errors.Wrapf(err, "getting %s %q in namespace %q", kind, name, s.namespace)
	}

	return true, obj.Object, nil
}
