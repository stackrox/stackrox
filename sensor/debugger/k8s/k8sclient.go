package k8s

import (
	"context"

	appVersioned "github.com/openshift/client-go/apps/clientset/versioned"
	configVersioned "github.com/openshift/client-go/config/clientset/versioned"
	routeVersioned "github.com/openshift/client-go/route/clientset/versioned"
	"github.com/pkg/errors"
	appsV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8sConfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

func MakeFakeClient() *ClientSet {
	return &ClientSet{
		k8s: fake.NewSimpleClientset(),
	}
}

type ClientSet struct {
	dynamic         dynamic.Interface
	k8s             kubernetes.Interface
	openshiftApps   appVersioned.Interface
	openshiftConfig configVersioned.Interface
	openshiftRoute  routeVersioned.Interface
}


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

func (c *ClientSet) SetupNamespace(name string) error {
	_, err := c.Kubernetes().CoreV1().Namespaces().Create(context.Background(), &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}, metav1.CreateOptions{})
	return err
}

func (c *ClientSet) SetupNginxDeployment(name string) error {
	_, err := c.Kubernetes().AppsV1().Deployments("default").Create(context.Background(), &appsV1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: appsV1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.14.2",
							Ports: []v1.ContainerPort{{ContainerPort: 80}},
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{})
	return err
}

func (c *ClientSet) SetupTestEnvironment() error {
	_, err := c.Kubernetes().CoreV1().Nodes().Create(context.Background(), &v1.Node{
		Spec: v1.NodeSpec{
			PodCIDR:       "",
			PodCIDRs:      nil,
			ProviderID:    "",
			Unschedulable: false,
			Taints:        nil,
		},
		Status: v1.NodeStatus{
			Capacity:        nil,
			Allocatable:     nil,
			Phase:           "",
			Conditions:      nil,
			Addresses:       nil,
			DaemonEndpoints: v1.NodeDaemonEndpoints{},
			NodeInfo:        v1.NodeSystemInfo{},
			Images:          nil,
			VolumesInUse:    nil,
			VolumesAttached: nil,
			Config:          nil,
		},
	}, metav1.CreateOptions{})
	return err
}

func (c *ClientSet) Kubernetes() kubernetes.Interface {
	return c.k8s
}

func (c *ClientSet) OpenshiftApps() appVersioned.Interface {
	return c.openshiftApps
}

func (c *ClientSet) OpenshiftConfig() configVersioned.Interface {
	return c.openshiftConfig
}

func (c *ClientSet) OpenshiftRoute() routeVersioned.Interface {
	return c.openshiftRoute
}

func (c *ClientSet) Dynamic() dynamic.Interface {
	return c.dynamic
}
