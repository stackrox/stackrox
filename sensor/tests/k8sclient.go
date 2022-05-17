package tests

import (
	"context"
	"testing"

	appVersioned "github.com/openshift/client-go/apps/clientset/versioned"
	configVersioned "github.com/openshift/client-go/config/clientset/versioned"
	routeVersioned "github.com/openshift/client-go/route/clientset/versioned"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func makeFakeClient() *clientSet {
	return &clientSet{
		k8s: fake.NewSimpleClientset(),
	}
}

type clientSet struct {
	dynamic         dynamic.Interface
	k8s             kubernetes.Interface
	openshiftApps   appVersioned.Interface
	openshiftConfig configVersioned.Interface
	openshiftRoute  routeVersioned.Interface
}

func (c *clientSet) setupTestEnvironment(t *testing.T) {
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
	require.NoError(t, err)
}

func (c *clientSet) Kubernetes() kubernetes.Interface {
	return c.k8s
}

func (c *clientSet) OpenshiftApps() appVersioned.Interface {
	return c.openshiftApps
}

func (c *clientSet) OpenshiftConfig() configVersioned.Interface {
	return c.openshiftConfig
}

func (c *clientSet) OpenshiftRoute() routeVersioned.Interface {
	return c.openshiftRoute
}

func (c *clientSet) Dynamic() dynamic.Interface {
	return c.dynamic
}
