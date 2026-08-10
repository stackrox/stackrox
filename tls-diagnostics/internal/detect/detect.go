package detect

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var (
	centralGVR = schema.GroupVersionResource{
		Group:    "platform.stackrox.io",
		Version:  "v1alpha1",
		Resource: "centrals",
	}
	securedClusterGVR = schema.GroupVersionResource{
		Group:    "platform.stackrox.io",
		Version:  "v1alpha1",
		Resource: "securedclusters",
	}
)

type Installation struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type Topology struct {
	Installations []Installation `json:"installations"`
	Summary       string         `json:"summary"`
}

func (t *Topology) Namespaces() []string {
	seen := make(map[string]bool)
	var ns []string
	for _, inst := range t.Installations {
		if !seen[inst.Namespace] {
			seen[inst.Namespace] = true
			ns = append(ns, inst.Namespace)
		}
	}
	return ns
}

func Detect(ctx context.Context, client dynamic.Interface) (*Topology, error) {
	topo := &Topology{}

	centrals, err := client.Resource(centralGVR).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing Central CRs: %w", err)
	}
	for _, item := range centrals.Items {
		topo.Installations = append(topo.Installations, Installation{
			Kind:      "Central",
			Name:      item.GetName(),
			Namespace: item.GetNamespace(),
		})
	}

	securedClusters, err := client.Resource(securedClusterGVR).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing SecuredCluster CRs: %w", err)
	}
	for _, item := range securedClusters.Items {
		topo.Installations = append(topo.Installations, Installation{
			Kind:      "SecuredCluster",
			Name:      item.GetName(),
			Namespace: item.GetNamespace(),
		})
	}

	topo.Summary = computeSummary(topo)
	return topo, nil
}

func computeSummary(topo *Topology) string {
	var hasCentral, hasSecuredCluster bool
	centralNS := make(map[string]bool)
	scNS := make(map[string]bool)

	for _, inst := range topo.Installations {
		switch inst.Kind {
		case "Central":
			hasCentral = true
			centralNS[inst.Namespace] = true
		case "SecuredCluster":
			hasSecuredCluster = true
			scNS[inst.Namespace] = true
		}
	}

	switch {
	case !hasCentral && !hasSecuredCluster:
		return "no StackRox operator installation found"
	case hasCentral && !hasSecuredCluster:
		return "just Central"
	case !hasCentral && hasSecuredCluster:
		return "just SecuredCluster"
	default:
		for ns := range centralNS {
			if scNS[ns] {
				return "both components, same namespace"
			}
		}
		return "both components, separate namespaces"
	}
}
