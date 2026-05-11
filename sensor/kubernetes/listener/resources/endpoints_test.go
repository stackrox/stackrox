package resources

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/sensor/common/service"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type countingNodeStore struct {
	getNodesCalls int
	nodes         []*nodeWrap
}

func (s *countingNodeStore) addOrUpdateNode(*nodeWrap) bool {
	return false
}

func (s *countingNodeStore) removeNode(*storage.Node) {}

func (s *countingNodeStore) getNode(string) *nodeWrap {
	return nil
}

func (s *countingNodeStore) getNodes() []*nodeWrap {
	s.getNodesCalls++
	return s.nodes
}

func TestEndpointDataForDeployment_ReusesFlattenedNodeIPsAcrossServices(t *testing.T) {
	serviceStore := newServiceStore()
	serviceStore.addOrUpdateService(wrapService(&v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc-1",
			Namespace: "stackrox",
			UID:       types.UID("svc-1"),
		},
		Spec: v1.ServiceSpec{
			Type:     v1.ServiceTypeNodePort,
			Selector: map[string]string{"app": "api"},
			Ports: []v1.ServicePort{{
				Name:       "http",
				Port:       80,
				NodePort:   30080,
				Protocol:   v1.ProtocolTCP,
				TargetPort: intstr.FromInt32(8080),
			}},
		},
	}))
	serviceStore.addOrUpdateService(wrapService(&v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc-2",
			Namespace: "stackrox",
			UID:       types.UID("svc-2"),
		},
		Spec: v1.ServiceSpec{
			Type:     v1.ServiceTypeNodePort,
			Selector: map[string]string{"app": "api"},
			Ports: []v1.ServicePort{{
				Name:       "metrics",
				Port:       81,
				NodePort:   30081,
				Protocol:   v1.ProtocolTCP,
				TargetPort: intstr.FromInt32(8081),
			}},
		},
	}))

	nodeStore := &countingNodeStore{
		nodes: []*nodeWrap{{
			addresses: []net.IPAddress{
				net.ParseIP("10.0.0.10"),
				net.ParseIP("10.0.0.11"),
			},
		}},
	}

	manager := newEndpointManager(serviceStore, newDeploymentStore(), newPodStore(), nodeStore, nil)
	data := manager.endpointDataForDeployment(&deploymentWrap{
		Deployment: &storage.Deployment{
			Id:        "deployment-1",
			Namespace: "stackrox",
			PodLabels: map[string]string{"app": "api"},
		},
		portConfigs: map[service.PortRef]*storage.PortConfig{},
	})

	if nodeStore.getNodesCalls != 1 {
		t.Fatalf("expected getNodes to be called once per deployment build, got %d", nodeStore.getNodesCalls)
	}
	if data == nil {
		t.Fatal("expected endpoint data to be returned")
	}
}
