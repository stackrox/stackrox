package tests

import "github.com/stackrox/rox/pkg/auth/permissions"

const (
	resourceAlert         = permissions.Resource("Alert")
	resourceCluster       = permissions.Resource("Cluster")
	resourceConfig        = permissions.Resource("Config")
	resourceDeployment    = permissions.Resource("Deployment")
	resourceImage         = permissions.Resource("Image")
	resourceInstallation  = permissions.Resource("InstallationInfo")
	resourceNetworkGraph  = permissions.Resource("NetworkGraph")
	resourceNetworkPolicy = permissions.Resource("NetworkPolicy")
	resourceNode          = permissions.Resource("Node")
	resourceRisk          = permissions.Resource("Risk")

	cluster1         = "Cluster1"
	cluster2         = "Cluster2"
	clusterClusterID = "clusterID"
	clusterCluster1  = "cluster1"
	clusterMyCluster = "mycluster"

	namespaceA   = "namespaceA"
	namespaceB   = "namespaceB"
	namespaceC   = "namespaceC"
	nsNamespace1 = "namespace1"
	nsNamespace2 = "namespace2"
	nsFoo        = "foo"
	nsBar        = "bar"
	nsBaz        = "baz"
	nsFar        = "far"
)
