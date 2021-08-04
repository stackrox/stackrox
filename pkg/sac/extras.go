package sac

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// EffectiveAccessScopeTreeExtras stores additional information for a tree node.
type EffectiveAccessScopeTreeExtras struct {
	ID     string
	Name   string
	Labels map[string]string
}

func extrasForCluster(cluster *storage.Cluster, detail v1.ComputeEffectiveAccessScopeRequest_Detail) EffectiveAccessScopeTreeExtras {
	extras := EffectiveAccessScopeTreeExtras{
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

func extrasForNamespace(namespace *storage.NamespaceMetadata, detail v1.ComputeEffectiveAccessScopeRequest_Detail) EffectiveAccessScopeTreeExtras {
	extras := EffectiveAccessScopeTreeExtras{
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
