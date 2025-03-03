package testconsts

import (
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
)

// Clusters and Namespaces for scoped access control tests
const (
	Cluster1     = fixtureconsts.Cluster1
	Cluster2     = fixtureconsts.Cluster2
	Cluster3     = fixtureconsts.Cluster3
	WrongCluster = fixtureconsts.ClusterFake1

	NamespaceA = "namespaceA"
	NamespaceB = "namespaceB"
	NamespaceC = "namespaceC"
)
