package effectiveaccessscope

import "github.com/stackrox/rox/generated/storage"

// ClusterForSAC is the minimum interface a cluster object has to satisfy
// to be used in effective access scope computation
type ClusterForSAC interface {
	GetID() string
	GetName() string
	GetLabels() map[string]string
}

// NamespaceForSAC is the minimum interface a namespace object has to satisfy
// to be used in effective access scope computation
type NamespaceForSAC interface {
	GetID() string
	GetName() string
	GetClusterID() string
	GetClusterName() string
	GetLabels() map[string]string
}

// region conversion from storage types

// StorageClusterToClusterForSAC converts a full storage.Cluster object
// to a minimal interface object implementing the ClusterForSAC interface.
func StorageClusterToClusterForSAC(cluster *storage.Cluster) ClusterForSAC {
	return &clusterForSAC{
		ID:     cluster.GetId(),
		Name:   cluster.GetName(),
		Labels: cluster.GetLabels(),
	}
}

// StorageNamespaceToNamespaceForSAC converts a full storage.NamespaceMetadata
// object to a minimal interface object implementing the NamespaceForSAC
// interface.
func StorageNamespaceToNamespaceForSAC(ns *storage.NamespaceMetadata) NamespaceForSAC {
	return &namespaceForSAC{
		ID:          ns.GetId(),
		Name:        ns.GetName(),
		ClusterID:   ns.GetClusterId(),
		ClusterName: ns.GetClusterName(),
		Labels:      ns.GetLabels(),
	}
}

// endregion conversion from storage types

// region interface implementations

type clusterForSAC struct {
	ID     string
	Name   string
	Labels map[string]string
}

func (c *clusterForSAC) GetID() string {
	if c == nil {
		return ""
	}
	return c.ID
}

func (c *clusterForSAC) GetName() string {
	if c == nil {
		return ""
	}
	return c.Name
}

func (c *clusterForSAC) GetLabels() map[string]string {
	if c == nil {
		return nil
	}
	return c.Labels
}

type namespaceForSAC struct {
	ID          string
	Name        string
	ClusterID   string
	ClusterName string
	Labels      map[string]string
}

func (n *namespaceForSAC) GetID() string {
	if n == nil {
		return ""
	}
	return n.ID
}

func (n *namespaceForSAC) GetName() string {
	if n == nil {
		return ""
	}
	return n.Name
}

func (n *namespaceForSAC) GetClusterID() string {
	if n == nil {
		return ""
	}
	return n.ClusterName
}

func (n *namespaceForSAC) GetClusterName() string {
	if n == nil {
		return ""
	}
	return n.ClusterName
}

func (n *namespaceForSAC) GetLabels() map[string]string {
	if n == nil {
		return nil
	}
	return n.Labels
}

// endregion interface implementations
