package fake

import (
	"math/rand"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	protocols  = [...]string{"TCP", "UDP", "SCTP"}
	ipFamilies = [...]string{"IPv4", "IPv6"}
)

func getRandProtocol() string {
	return protocols[rand.Intn(len(protocols))]
}

func getRandPort() uint32 {
	return rand.Uint32() % 63556
}

func getIPFamily() string {
	return ipFamilies[rand.Intn(len(ipFamilies))]
}

func getClusterIP() *v1.Service {
	ns := namespacesWithDeploymentsPool.mustGetRandomElem()
	labels := labelsPool.randomElem(ns)
	clusterIP := generateIP()
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      randStringWithLength(16),
			Namespace: ns,
			UID:       newUUID(),
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

func getNodePort() *v1.Service {
	ns := namespacesWithDeploymentsPool.mustGetRandomElem()
	labels := labelsPool.randomElem(ns)
	clusterIP := generateIP()
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      randStringWithLength(16),
			Namespace: ns,
			UID:       newUUID(),
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
					NodePort: int32(getRandPort()),
				},
			},
			ClusterIP:  clusterIP,
			ClusterIPs: []string{clusterIP},
			IPFamilies: []v1.IPFamily{v1.IPFamily(getIPFamily())},
		},
	}
}

func getLoadBalancer() *v1.Service {
	ns := namespacesWithDeploymentsPool.mustGetRandomElem()
	labels := labelsPool.randomElem(ns)
	clusterIP := generateIP()
	internalTrafficPolicy := v1.ServiceInternalTrafficPolicyCluster
	allocateLoadBalancerNodePorts := true
	ipFamilyPolicy := v1.IPFamilyPolicySingleStack
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      randStringWithLength(16),
			Namespace: ns,
			UID:       newUUID(),
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
					NodePort: int32(getRandPort()),
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

func getService(workload ServiceWorkload) []runtime.Object {
	objects := make([]runtime.Object, 0, workload.NumClusterIPs+workload.NumNodePorts+workload.NumLoadBalancers)
	for i := 0; i < workload.NumClusterIPs; i++ {
		clusterIP := getClusterIP()
		objects = append(objects, clusterIP)
	}
	for i := 0; i < workload.NumNodePorts; i++ {
		nodePort := getNodePort()
		objects = append(objects, nodePort)
	}
	for i := 0; i < workload.NumLoadBalancers; i++ {
		loadBalancer := getLoadBalancer()
		objects = append(objects, loadBalancer)
	}
	return objects
}
