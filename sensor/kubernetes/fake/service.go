package fake

import (
	"math/rand"

	"github.com/stackrox/rox/pkg/sync"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	protocols  = [...]string{"TCP", "UDP", "SCTP"}
	ipFamilies = [...]string{"IPv4", "IPv6"}
	// nextNodePortMutex is used to synchronize access to the nextNodePort variable,
	// despite it not being used concurrently at the moment. This is added to
	// protect against future concurrent access (and to silent AI reviewers).
	nextNodePortMutex       = &sync.Mutex{}
	nextNodePort      int32 = 30000
)

func getRandProtocol() string {
	return protocols[rand.Intn(len(protocols))]
}

func getRandPort() uint32 {
	return rand.Uint32() % 63556
}

// getPseudUniqueNodePort returns successive ports in the Kubernetes NodePort range
// [30000, 32767]. The counter wraps around after 2768 allocations, which matches
// the real-cluster ceiling for this range.
func getPseudUniqueNodePort() int32 {
	nextNodePortMutex.Lock()
	defer nextNodePortMutex.Unlock()
	port := nextNodePort
	nextNodePort++
	if nextNodePort > 32767 {
		nextNodePort = 30000
	}
	return port
}

func getIPFamily() string {
	return ipFamilies[rand.Intn(len(ipFamilies))]
}

func getClusterIP(id string, lblPool *labelsPoolPerNamespace) *v1.Service {
	ns := namespacesWithDeploymentsPool.mustGetRandomElem()
	labels := lblPool.randomElem(ns)
	clusterIP := generateIP()
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      randStringWithLength(16),
			Namespace: ns,
			UID:       idOrNewUID(id),
		},
		Spec: v1.ServiceSpec{
			Type:     v1.ServiceTypeClusterIP,
			Selector: labels,
			Ports: []v1.ServicePort{
				{
					Protocol: v1.Protocol(getRandProtocol()),
					Port:     int32(getRandPort()),
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: int32(getRandPort()),
					},
				},
			},
			ClusterIP:  clusterIP,
			ClusterIPs: []string{clusterIP},
			IPFamilies: []v1.IPFamily{v1.IPFamily(getIPFamily())},
		},
	}
}

func getNodePort(id string, lblPool *labelsPoolPerNamespace) *v1.Service {
	ns := namespacesWithDeploymentsPool.mustGetRandomElem()
	labels := lblPool.randomElem(ns)
	clusterIP := generateIP()
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      randStringWithLength(16),
			Namespace: ns,
			UID:       idOrNewUID(id),
		},
		Spec: v1.ServiceSpec{
			Type:     v1.ServiceTypeNodePort,
			Selector: labels,
			Ports: []v1.ServicePort{
				{
					Protocol: v1.Protocol(getRandProtocol()),
					Port:     int32(getRandPort()),
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: int32(getRandPort()),
					},
					NodePort: getPseudUniqueNodePort(),
				},
			},
			ClusterIP:  clusterIP,
			ClusterIPs: []string{clusterIP},
			IPFamilies: []v1.IPFamily{v1.IPFamily(getIPFamily())},
		},
	}
}

func getLoadBalancer(id string, lblPool *labelsPoolPerNamespace) *v1.Service {
	ns := namespacesWithDeploymentsPool.mustGetRandomElem()
	labels := lblPool.randomElem(ns)
	clusterIP := generateIP()
	internalTrafficPolicy := v1.ServiceInternalTrafficPolicyCluster
	allocateLoadBalancerNodePorts := true
	ipFamilyPolicy := v1.IPFamilyPolicySingleStack
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      randStringWithLength(16),
			Namespace: ns,
			UID:       idOrNewUID(id),
		},
		Spec: v1.ServiceSpec{
			Type:     v1.ServiceTypeLoadBalancer,
			Selector: labels,
			Ports: []v1.ServicePort{
				{
					Protocol: v1.Protocol(getRandProtocol()),
					Port:     int32(getRandPort()),
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: int32(getRandPort()),
					},
					NodePort: getPseudUniqueNodePort(),
				},
			},
			ClusterIP:                     clusterIP,
			ClusterIPs:                    []string{clusterIP},
			IPFamilies:                    []v1.IPFamily{v1.IPFamily(getIPFamily())},
			ExternalTrafficPolicy:         v1.ServiceExternalTrafficPolicyTypeCluster,
			InternalTrafficPolicy:         &internalTrafficPolicy,
			AllocateLoadBalancerNodePorts: &allocateLoadBalancerNodePorts,
			PublishNotReadyAddresses:      false,
			IPFamilyPolicy:                &ipFamilyPolicy,
		},
		Status: v1.ServiceStatus{
			LoadBalancer: v1.LoadBalancerStatus{
				Ingress: []v1.LoadBalancerIngress{
					{
						IP: generateIP(),
					},
				},
			},
		},
	}
}

func (w *WorkloadManager) getServices(workload ServiceWorkload, ids []string) []runtime.Object {
	objects := make([]runtime.Object, 0, workload.NumClusterIPs+workload.NumNodePorts+workload.NumLoadBalancers)
	for i := 0; i < workload.NumClusterIPs; i++ {
		clusterIP := getClusterIP(getID(ids, i), w.labelsPool)
		w.writeID(servicePrefix, clusterIP.UID)
		objects = append(objects, clusterIP)
	}
	for i := 0; i < workload.NumNodePorts; i++ {
		nodePort := getNodePort(getID(ids, i+workload.NumClusterIPs), w.labelsPool)
		w.writeID(servicePrefix, nodePort.UID)
		objects = append(objects, nodePort)
	}
	for i := 0; i < workload.NumLoadBalancers; i++ {
		loadBalancer := getLoadBalancer(getID(ids, i+workload.NumClusterIPs+workload.NumNodePorts), w.labelsPool)
		w.writeID(servicePrefix, loadBalancer.UID)
		objects = append(objects, loadBalancer)
	}
	return objects
}
