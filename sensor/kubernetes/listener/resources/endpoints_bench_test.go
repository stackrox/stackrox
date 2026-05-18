package resources

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/sensor/common/service"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func BenchmarkEndpointDataForDeployment_NodePortServices(b *testing.B) {
	for _, tc := range []struct {
		name        string
		numServices int
		numNodes    int
	}{
		{name: "services_10_nodes_10", numServices: 10, numNodes: 10},
		{name: "services_100_nodes_10", numServices: 100, numNodes: 10},
		{name: "services_100_nodes_100", numServices: 100, numNodes: 100},
	} {
		b.Run(tc.name, func(b *testing.B) {
			manager, deployment := benchmarkEndpointDataForDeploymentSetup(tc.numServices, tc.numNodes)
			b.ReportAllocs()

			for b.Loop() {
				_ = manager.endpointDataForDeployment(deployment)
			}
		})
	}
}

func benchmarkEndpointDataForDeploymentSetup(numServices, numNodes int) (*endpointManagerImpl, *deploymentWrap) {
	serviceStore := newServiceStore()
	nodeStore := newNodeStore()

	labels := map[string]string{"app": "api"}
	for i := range numServices {
		serviceStore.addOrUpdateService(wrapService(&v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("svc-%d", i),
				Namespace: "stackrox",
				UID:       types.UID(fmt.Sprintf("svc-%d", i)),
			},
			Spec: v1.ServiceSpec{
				Type:      v1.ServiceTypeNodePort,
				ClusterIP: benchmarkIPv4(10, i),
				Selector:  labels,
				Ports: []v1.ServicePort{{
					Name:       fmt.Sprintf("port-%d", i),
					Port:       80,
					NodePort:   int32(30_000 + i),
					Protocol:   v1.ProtocolTCP,
					TargetPort: intstr.FromInt32(int32(8_080 + i)),
				}},
			},
		}))
	}

	for i := range numNodes {
		nodeStore.addOrUpdateNode(&nodeWrap{
			Node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("node-%d", i),
				},
			},
			addresses: []net.IPAddress{
				net.ParseIP(benchmarkIPv4(11, i*2)),
				net.ParseIP(benchmarkIPv4(12, i*2+1)),
			},
		})
	}

	manager := newEndpointManager(serviceStore, newDeploymentStore(), newPodStore(), nodeStore, nil)
	deployment := &deploymentWrap{
		Deployment: &storage.Deployment{
			Id:        "deployment-1",
			Namespace: "stackrox",
			PodLabels: labels,
		},
		portConfigs: map[service.PortRef]*storage.PortConfig{},
	}

	return manager, deployment
}

func benchmarkIPv4(firstOctet, idx int) string {
	idx++
	return fmt.Sprintf("%d.%d.%d.%d", firstOctet, (idx/65_536)%256, (idx/256)%256, idx%256)
}
