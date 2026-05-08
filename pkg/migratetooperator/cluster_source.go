package migratetooperator

import (
	"context"
	"time"

	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	// Activate auth providers for client-go.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

const k8sRequestTimeout = 30 * time.Second

type clusterSource struct {
	typed     kubernetes.Interface
	dynamic   dynamic.Interface
	namespace string
}

// NewClusterSource creates a Source that reads resources from a live Kubernetes cluster.
func NewClusterSource(namespace string) (*clusterSource, error) {
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

func (s *clusterSource) Deployment(name string) (*appsv1.Deployment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), k8sRequestTimeout)
	defer cancel()

	dep, err := s.typed.AppsV1().Deployments(s.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "getting Deployment %q in namespace %q", name, s.namespace)
	}
	return dep, nil
}

func (s *clusterSource) Service(name string) (*corev1.Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), k8sRequestTimeout)
	defer cancel()

	svc, err := s.typed.CoreV1().Services(s.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "getting Service %q in namespace %q", name, s.namespace)
	}
	return svc, nil
}

func (s *clusterSource) Secret(name string) (*corev1.Secret, error) {
	ctx, cancel := context.WithTimeout(context.Background(), k8sRequestTimeout)
	defer cancel()

	secret, err := s.typed.CoreV1().Secrets(s.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "getting Secret %q in namespace %q", name, s.namespace)
	}
	return secret, nil
}

var routeGVR = schema.GroupVersionResource{Group: "route.openshift.io", Version: "v1", Resource: "routes"}

func (s *clusterSource) Route(name string) (*unstructured.Unstructured, error) {
	ctx, cancel := context.WithTimeout(context.Background(), k8sRequestTimeout)
	defer cancel()

	obj, err := s.dynamic.Resource(routeGVR).Namespace(s.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "getting Route %q in namespace %q", name, s.namespace)
	}
	return obj, nil
}

func (s *clusterSource) DaemonSet(name string) (*appsv1.DaemonSet, error) {
	ctx, cancel := context.WithTimeout(context.Background(), k8sRequestTimeout)
	defer cancel()

	ds, err := s.typed.AppsV1().DaemonSets(s.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "getting DaemonSet %q in namespace %q", name, s.namespace)
	}
	return ds, nil
}

func (s *clusterSource) ValidatingWebhookConfiguration(name string) (*admissionv1.ValidatingWebhookConfiguration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), k8sRequestTimeout)
	defer cancel()

	vwc, err := s.typed.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "getting ValidatingWebhookConfiguration %q", name)
	}
	return vwc, nil
}
