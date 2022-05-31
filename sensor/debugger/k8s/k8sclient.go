package k8s

import (
	"context"
	"testing"

	appVersioned "github.com/openshift/client-go/apps/clientset/versioned"
	configVersioned "github.com/openshift/client-go/config/clientset/versioned"
	routeVersioned "github.com/openshift/client-go/route/clientset/versioned"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8sConfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

// MakeFakeClient creates a k8s client that is not connected to any cluster
func MakeFakeClient() *ClientSet {
	return &ClientSet{
		k8s: fake.NewSimpleClientset(),
	}
}

// ClientSet is a test version of kubernetes.ClientSet
type ClientSet struct {
	dynamic         dynamic.Interface
	k8s             kubernetes.Interface
	openshiftApps   appVersioned.Interface
	openshiftConfig configVersioned.Interface
	openshiftRoute  routeVersioned.Interface
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

// Dynamic returns the Dynamic interface
// This is not used in tests!
func (c *ClientSet) Dynamic() dynamic.Interface {
	return c.dynamic
}

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

func (c *ClientSet) ResetDeployments(t *testing.T) {
	deletionPropagation := metav1.DeletionPropagation("foreground")
	err := c.k8s.AppsV1().Deployments("default").DeleteCollection(context.Background(), metav1.DeleteOptions{
		TypeMeta:           metav1.TypeMeta{},
		PropagationPolicy:  &deletionPropagation,
	}, metav1.ListOptions{})
	require.NoError(t, err)

	err = c.k8s.CoreV1().Namespaces().Delete(context.Background(), "default", metav1.DeleteOptions{
		TypeMeta:           metav1.TypeMeta{},
		PropagationPolicy:  &deletionPropagation,
	})
	require.NoError(t, err)
	_, err = c.k8s.CoreV1().Namespaces().Create(context.Background(), &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
		Spec:   v1.NamespaceSpec{},
		Status: v1.NamespaceStatus{},
	}, metav1.CreateOptions{})
}

func (c *ClientSet) MustCreateRole(t *testing.T, name string, ) {
	_, err := c.k8s.RbacV1().Roles("default").Create(context.Background(), &v12.Role{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Rules: []v12.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{""},
			Verbs:     []string{"get"},
		}, {
			APIGroups: []string{""},
			Resources: []string{""},
			Verbs:     []string{"list"},
		}},
	}, metav1.CreateOptions{})
	require.NoError(t, err)
}

func (c *ClientSet) MustCreateRoleBinding(t *testing.T, bindingName, roleName, serviceAccountName string ) {
	_, err := c.k8s.RbacV1().RoleBindings("default").Create(context.Background(), &v12.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:                       bindingName,
			Namespace:                  "default",
		},
		Subjects:   []v12.Subject{
			{
				Name:      serviceAccountName,
				Kind: "ServiceAccount",
				Namespace: "default",
			},
		},
		RoleRef:    v12.RoleRef{
			APIGroup: "",
			Kind:     "",
			Name:     roleName,
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)
}

type DeploymentOpts func(obj *appsv1.Deployment)

func WithServiceAccountName(name string) DeploymentOpts {
	return func(obj *appsv1.Deployment) {
		obj.Spec.Template.Spec.ServiceAccountName = name
	}
}

func (c *ClientSet) MustCreateDeployment(t *testing.T, name string, opts ...DeploymentOpts) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nginx",
				},
				Spec: v1.PodSpec{
					Volumes:        nil,
					InitContainers: nil,
					Containers: []v1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.14.2",
							Ports: []v1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{},
	}

	for _, opt := range opts {
		opt(deployment)
	}

	_, err := c.k8s.AppsV1().Deployments("default").Create(context.Background(), deployment, metav1.CreateOptions{})
	require.NoError(t, err)
}
