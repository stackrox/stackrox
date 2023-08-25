package k8s

import (
	"context"
	"log"
	"testing"

	appVersioned "github.com/openshift/client-go/apps/clientset/versioned"
	configVersioned "github.com/openshift/client-go/config/clientset/versioned"
	operatorVersioned "github.com/openshift/client-go/operator/clientset/versioned"
	routeVersioned "github.com/openshift/client-go/route/clientset/versioned"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	k8sConfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

// MakeFakeClient creates a k8s client that is not connected to any cluster
func MakeFakeClient() *ClientSet {
	return &ClientSet{
		k8s: fake.NewSimpleClientset(),
	}
}

// MakeFakeClientFromRest creates a k8s client from rest.Config
func MakeFakeClientFromRest(restConfig *rest.Config) *ClientSet {
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Panicf("Creating Kubernetes clientset: %v", err)
	}

	return &ClientSet{
		k8s: client,
	}
}

// ClientSet is a test version of kubernetes.ClientSet
type ClientSet struct {
	dynamic           dynamic.Interface
	k8s               kubernetes.Interface
	openshiftApps     appVersioned.Interface
	openshiftConfig   configVersioned.Interface
	openshiftRoute    routeVersioned.Interface
	openshiftOperator operatorVersioned.Interface
}

// MakeOutOfClusterClient creates a k8s client that uses host configuration to connect to a cluster.
// If host machine has a KUBECONFIG env set it will use it to connect to the respective cluster.
func MakeOutOfClusterClient() (*ClientSet, error) {
	config, err := k8sConfig.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "getting k8s config")
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "creating ClientSet")
	}

	return &ClientSet{
		k8s: k8sClient,
	}, nil
}

// Kubernetes returns the kubernetes interface
func (c *ClientSet) Kubernetes() kubernetes.Interface {
	return c.k8s
}

// OpenshiftApps returns the OpenshiftApps interface
// This is not used in tests!
func (c *ClientSet) OpenshiftApps() appVersioned.Interface {
	return c.openshiftApps
}

// OpenshiftConfig returns the OpenshiftConfig interface
// This is not used in tests!
func (c *ClientSet) OpenshiftConfig() configVersioned.Interface {
	return c.openshiftConfig
}

// OpenshiftRoute returns the OpenshiftRoute interface
// This is not used in tests!
func (c *ClientSet) OpenshiftRoute() routeVersioned.Interface {
	return c.openshiftRoute
}

// OpenshiftOperator returns the OpenshiftOperator interface
// This is not used in tests!
func (c *ClientSet) OpenshiftOperator() operatorVersioned.Interface {
	return c.openshiftOperator
}

// Dynamic returns the Dynamic interface
// This is not used in tests!
func (c *ClientSet) Dynamic() dynamic.Interface {
	return c.dynamic
}

// SetupExampleCluster creates a fake node and default namespace in the fake k8s client.
func (c *ClientSet) SetupExampleCluster(t *testing.T) {
	_, err := c.k8s.CoreV1().Nodes().Create(context.Background(), &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "go-test-node",
		},
		Spec:   v1.NodeSpec{},
		Status: v1.NodeStatus{},
	}, metav1.CreateOptions{})

	require.NoError(t, err)

	_, err = c.k8s.CoreV1().Namespaces().Create(context.Background(), &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
		Spec:   v1.NamespaceSpec{},
		Status: v1.NamespaceStatus{},
	}, metav1.CreateOptions{})

	require.NoError(t, err)
}
