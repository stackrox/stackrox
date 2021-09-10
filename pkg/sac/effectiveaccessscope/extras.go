package effectiveaccessscope

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// ScopeTreeExtras stores additional information for a tree node.
type ScopeTreeExtras struct {
	ID     string
	Name   string
	Labels map[string]string
}

func extrasForCluster(cluster *storage.Cluster, detail v1.ComputeEffectiveAccessScopeRequest_Detail) ScopeTreeExtras {
	extras := ScopeTreeExtras{
		ID: cluster.GetId(),
	}
	if detail != v1.ComputeEffectiveAccessScopeRequest_MINIMAL {
		extras.Name = cluster.GetName()
	}
	if detail == v1.ComputeEffectiveAccessScopeRequest_HIGH {
		extras.Labels = cluster.GetLabels()
	}
	return extras
}

func extrasForNamespace(namespace *storage.NamespaceMetadata, detail v1.ComputeEffectiveAccessScopeRequest_Detail) ScopeTreeExtras {
	extras := ScopeTreeExtras{
		ID: namespace.GetId(),
	}
	if detail != v1.ComputeEffectiveAccessScopeRequest_MINIMAL {
		extras.Name = namespace.GetName()
	}
	if detail == v1.ComputeEffectiveAccessScopeRequest_HIGH {
		extras.Labels = namespace.GetLabels()
	}
	return extras
}
