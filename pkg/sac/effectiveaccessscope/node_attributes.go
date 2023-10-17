package effectiveaccessscope

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
)

// treeNodeAttributes stores additional information for a tree node.
type treeNodeAttributes struct {
	ID     string
	Name   string
	Labels map[string]string
}

func (t *treeNodeAttributes) copy() *treeNodeAttributes {
	labels := make(map[string]string, len(t.Labels))
	for k, v := range t.Labels {
		labels[k] = v
	}
	return &treeNodeAttributes{
		ID:     t.ID,
		Name:   t.Name,
		Labels: labels,
	}
}

func nodeAttributesForCluster(cluster ClusterForSAC, detail v1.ComputeEffectiveAccessScopeRequest_Detail) treeNodeAttributes {
	attributes := treeNodeAttributes{
		ID: cluster.GetID(),
	}
	if detail != v1.ComputeEffectiveAccessScopeRequest_MINIMAL {
		attributes.Name = cluster.GetName()
	}
	if detail == v1.ComputeEffectiveAccessScopeRequest_HIGH {
		attributes.Labels = cluster.GetLabels()
	}
	return attributes
}

func nodeAttributesForNamespace(namespace NamespaceForSAC, detail v1.ComputeEffectiveAccessScopeRequest_Detail) treeNodeAttributes {
	attributes := treeNodeAttributes{
		ID: namespace.GetID(),
	}
	if detail != v1.ComputeEffectiveAccessScopeRequest_MINIMAL {
		attributes.Name = namespace.GetName()
	}
	if detail == v1.ComputeEffectiveAccessScopeRequest_HIGH {
		attributes.Labels = namespace.GetLabels()
	}
	return attributes
}
