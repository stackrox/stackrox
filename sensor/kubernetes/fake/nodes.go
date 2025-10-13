package fake

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (w *WorkloadManager) getNodes(workload NodeWorkload, ids []string) []*corev1.Node {
	nodes := make([]*corev1.Node, 0, workload.NumNodes)
	for i := 0; i < workload.NumNodes; i++ {
		name := fmt.Sprintf("gke-setup-devadda2-large-pool-a9523a88-%d", i)
		node := &corev1.Node{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Node",
			},
			ObjectMeta: metav1.ObjectMeta{
				UID:  idOrNewUID(getID(ids, i)),
				Name: name,
				Annotations: map[string]string{
					"container.googleapis.com/instance_id":                   "6748817401527894896",
					"node.alpha.kubernetes.io/ttl":                           "0",
					"volumes.kubernetes.io/controller-managed-attach-detach": "true",
				},
				Labels: map[string]string{
					"beta.kubernetes.io/arch":                  "amd64",
					"beta.kubernetes.io/fluentd-ds-ready":      "true",
					"beta.kubernetes.io/instance-type":         "e2-standard-8",
					"beta.kubernetes.io/masq-agent-ds-ready":   "true",
					"beta.kubernetes.io/os":                    "linux",
					"cloud.google.com/gke-nodepool":            "large-pool",
					"cloud.google.com/gke-os-distribution":     "ubuntu",
					"failure-domain.beta.kubernetes.io/region": "us-west1",
					"failure-domain.beta.kubernetes.io/zone":   "us-west1-c",
					"kubernetes.io/arch":                       "amd64",
					"kubernetes.io/hostname":                   name,
					"kubernetes.io/os":                         "linux",
					"node.kubernetes.io/masq-agent-ds-ready":   "true",
					"projectcalico.org/ds-ready":               "true",
				},
			},
			Spec: corev1.NodeSpec{
				PodCIDR: "10.12.4.0/24",
				Taints: []corev1.Taint{
					{
						Key:   "key",
						Value: "value",
					},
				},
			},
			Status: corev1.NodeStatus{
				Addresses: []corev1.NodeAddress{
					{
						Address: "10.138.28.6",
						Type:    "InternalIP",
					},
					{
						Address: "35.185.217.58",
						Type:    "ExternalIP",
					},
					{
						Address: fmt.Sprintf("%s.c.ultra-current-825.internal", name),
						Type:    "InternalDNS",
					},
					{
						Address: fmt.Sprintf("%s.c.ultra-current-825.internal", name),
						Type:    "Hostname",
					},
				},
				NodeInfo: corev1.NodeSystemInfo{
					KernelVersion:           "4.15.0-1044-gke",
					OSImage:                 "Ubuntu 18.04.3 LTS",
					ContainerRuntimeVersion: "docker://18.9.7",
					KubeletVersion:          "v1.14.10-gke.27",
					KubeProxyVersion:        "v1.14.10-gke.27",
					OperatingSystem:         "linux",
					Architecture:            "amd64",
				},
			},
		}
		nodes = append(nodes, node)
	}
	return nodes
}
