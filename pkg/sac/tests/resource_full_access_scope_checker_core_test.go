package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/assert"
)

func getSampleTestScopeCheckerCore(t *testing.T) sac.ScopeCheckerCore {
	return sac.TestScopeCheckerCoreFromFullScopeMap(t, sac.TestScopeMap{
		storage.Access_READ_ACCESS: {
			resourceCluster: &sac.TestResourceScope{Clusters: map[string]*sac.TestClusterScope{
				cluster1: {Included: true},
				cluster2: {Namespaces: []string{namespaceC}},
			}},
			resourceDeployment: &sac.TestResourceScope{Clusters: map[string]*sac.TestClusterScope{
				cluster1: {Namespaces: []string{namespaceA, namespaceC}},
			}},
			resourceNetworkGraph: &sac.TestResourceScope{Included: true},
			resourceNetworkPolicy: &sac.TestResourceScope{Clusters: map[string]*sac.TestClusterScope{
				cluster1: {Namespaces: []string{namespaceB}},
				cluster2: {Namespaces: []string{namespaceC}},
			}},
			resourceNode: &sac.TestResourceScope{Included: true},
		},
		storage.Access_READ_WRITE_ACCESS: {
			resourceNetworkPolicy: &sac.TestResourceScope{Clusters: map[string]*sac.TestClusterScope{
				cluster2: {Namespaces: []string{namespaceC}},
			}},
		},
	})
}

var (
	sampleCompactTreeClusterRead = effectiveaccessscope.ScopeTreeCompacted{
		cluster1: {"*"},
		cluster2: {namespaceC},
	}
	sampleCompactTreeDeploymentRead = effectiveaccessscope.ScopeTreeCompacted{
		cluster1: {namespaceA, namespaceC},
	}
	sampleCompactTreeNetworkPolicyRead = effectiveaccessscope.ScopeTreeCompacted{
		cluster1: {namespaceB},
		cluster2: {namespaceC},
	}
	sampleCompactTreeNetworkPolicyWrite = effectiveaccessscope.ScopeTreeCompacted{
		cluster2: {namespaceC},
	}
)

type drillDown struct {
	key     sac.ScopeKey
	allowed bool
}

func TestUnrestrictedResourceReadSubSCCAllowed(t *testing.T) {
	testCases := map[string]struct {
		baseSCCDrillDown    []drillDown
		wrapperResource     permissions.Resource
		wrappedSCCDrillDown []drillDown
	}{
		"Test Unrestricted Image Read, checking read on Cluster for cluster1": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceCluster)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
			},
			wrapperResource: resourceImage,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceCluster)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
			},
		},
		"Test Unrestricted Image Read, checking read on Deployment for cluster1 then namespaceA (allowed)": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceA)},
			},
			wrapperResource: resourceImage,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceA)},
			},
		},
		"Test Unrestricted Image Read, checking read on Deployment for cluster1 then namespaceB (denied)": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceB)},
			},
			wrapperResource: resourceImage,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceB)},
			},
		},
		"Test Unrestricted Image Read, checking read on Image for cluster1 then namespaceA": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceImage)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceA)},
			},
			wrapperResource: resourceImage,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceImage)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceA)},
			},
		},
		"Test Unrestricted Image Read, checking write on Image for cluster1 then namespaceA": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceImage)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceA)},
			},
			wrapperResource: resourceImage,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceImage)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceA)},
			},
		},
		"Test Unrestricted Deployment Read (override), checking read on Deployment for cluster1, then namespaceA": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceA)},
			},
			wrapperResource: resourceDeployment,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceA)},
			},
		},
		"Test Unrestricted Deployment Read (override), checking read on Deployment for cluster1, then namespaceB": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceB)},
			},
			wrapperResource: resourceDeployment,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceB)},
			},
		},
		"Test Unrestricted Deployment Read (override), checking read on Deployment for cluster2, then namespaceB": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster2)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceB)},
			},
			wrapperResource: resourceDeployment,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: true, key: sac.ClusterScopeKey(cluster2)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceB)},
			},
		},
		"Test Unrestricted Deployment Read (override), checking write on Deployment for cluster1, then namespaceA": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceA)},
			},
			wrapperResource: resourceDeployment,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceA)},
			},
		},
		"Test Unrestricted NetworkGraph Read (override already fully accessible), checking read on NetworkGraph for cluster1, then namespaceA": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceNetworkGraph)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceA)},
			},
			wrapperResource: resourceNetworkGraph,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceNetworkGraph)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceA)},
			},
		},
		"Test Unrestricted NetworkGraph Read (override already fully accessible), checking read on NetworkGraph for cluster1, then namespaceB": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceNetworkGraph)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceB)},
			},
			wrapperResource: resourceNetworkGraph,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceNetworkGraph)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceB)},
			},
		},
		"Test Unrestricted NetworkGraph Read (override already fully accessible), checking read on NetworkGraph for cluster2, then namespaceB": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceNetworkGraph)},
				{allowed: true, key: sac.ClusterScopeKey(cluster2)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceB)},
			},
			wrapperResource: resourceNetworkGraph,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceNetworkGraph)},
				{allowed: true, key: sac.ClusterScopeKey(cluster2)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceB)},
			},
		},
	}

	baseTestCtx := sac.WithGlobalAccessScopeChecker(context.Background(), getSampleTestScopeCheckerCore(t))
	for testName, tc := range testCases {
		t.Run(testName, func(it *testing.T) {
			// Validate access drill down on base SCC
			checker := sac.GlobalAccessScopeChecker(baseTestCtx)
			assert.False(it, checker.IsAllowed())
			for _, down := range tc.baseSCCDrillDown {
				fmt.Println("Drilling down to", down.key.String())
				checker = checker.SubScopeChecker(down.key)
				assert.Equal(it, down.allowed, checker.IsAllowed())
			}
			resourceMD, found := resources.MetadataForResource(tc.wrapperResource)
			assert.True(it, found)
			wrappedCtx := sac.WithUnrestrictedResourceRead(baseTestCtx, resourceMD)
			wrappedChecker := sac.GlobalAccessScopeChecker(wrappedCtx)
			assert.False(it, wrappedChecker.IsAllowed())
			for _, down := range tc.wrappedSCCDrillDown {
				fmt.Println("Wrapped - Drilling down to", down.key.String())
				wrappedChecker = wrappedChecker.SubScopeChecker(down.key)
				assert.Equal(it, down.allowed, wrappedChecker.IsAllowed())
			}
		})
	}
}

func TestUnrestrictedResourceReadWriteSubSCCAllowed(t *testing.T) {
	testCases := map[string]struct {
		baseSCCDrillDown    []drillDown
		wrapperResource     permissions.Resource
		wrappedSCCDrillDown []drillDown
	}{
		"Test Unrestricted Image ReadWrite, checking read on Cluster for cluster1": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceCluster)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
			},
			wrapperResource: resourceImage,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceCluster)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
			},
		},
		"Test Unrestricted Image ReadWrite, checking write on Cluster for cluster1": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceCluster)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
			},
			wrapperResource: resourceImage,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceCluster)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
			},
		},
		"Test Unrestricted Image ReadWrite, checking read on Deployment for cluster1 then namespaceA (allowed)": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceA)},
			},
			wrapperResource: resourceImage,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceA)},
			},
		},
		"Test Unrestricted Image ReadWrite, checking write on Deployment for cluster1 then namespaceA (allowed)": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceA)},
			},
			wrapperResource: resourceImage,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceA)},
			},
		},
		"Test Unrestricted Image ReadWrite, checking read on Deployment for cluster1 then namespaceB (denied)": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceB)},
			},
			wrapperResource: resourceImage,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceB)},
			},
		},
		"Test Unrestricted Image ReadWrite, checking write on Deployment for cluster1 then namespaceB (denied)": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceB)},
			},
			wrapperResource: resourceImage,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceB)},
			},
		},
		"Test Unrestricted Image ReadWrite, checking read on Image for cluster1 then namespaceA": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceImage)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceA)},
			},
			wrapperResource: resourceImage,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceImage)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceA)},
			},
		},
		"Test Unrestricted Image ReadWrite, checking write on Image for cluster1 then namespaceA": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceImage)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceA)},
			},
			wrapperResource: resourceImage,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceImage)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceA)},
			},
		},
		"Test Unrestricted Deployment ReadWrite (override), checking read on Deployment for cluster1, then namespaceA": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceA)},
			},
			wrapperResource: resourceDeployment,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceA)},
			},
		},
		"Test Unrestricted Deployment ReadWrite (override), checking write on Deployment for cluster1, then namespaceA": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceA)},
			},
			wrapperResource: resourceDeployment,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceA)},
			},
		},
		"Test Unrestricted Deployment ReadWrite (override), checking read on Deployment for cluster1, then namespaceB": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceB)},
			},
			wrapperResource: resourceDeployment,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceB)},
			},
		},
		"Test Unrestricted Deployment ReadWrite (override), checking write on Deployment for cluster1, then namespaceB": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceB)},
			},
			wrapperResource: resourceDeployment,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceB)},
			},
		},
		"Test Unrestricted Deployment ReadWrite (override), checking read on Deployment for cluster2, then namespaceB": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster2)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceB)},
			},
			wrapperResource: resourceDeployment,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: true, key: sac.ClusterScopeKey(cluster2)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceB)},
			},
		},
		"Test Unrestricted Deployment ReadWrite (override), checking write on Deployment for cluster2, then namespaceB": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: false, key: sac.ClusterScopeKey(cluster2)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceB)},
			},
			wrapperResource: resourceDeployment,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceDeployment)},
				{allowed: true, key: sac.ClusterScopeKey(cluster2)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceB)},
			},
		},
		"Test Unrestricted NetworkGraph ReadWrite (override already fully readable), checking read on NetworkGraph for cluster1, then namespaceA": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceNetworkGraph)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceA)},
			},
			wrapperResource: resourceNetworkGraph,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceNetworkGraph)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceA)},
			},
		},
		"Test Unrestricted NetworkGraph ReadWrite (override already fully readable), checking write on NetworkGraph for cluster1, then namespaceA": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceNetworkGraph)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceA)},
			},
			wrapperResource: resourceNetworkGraph,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceNetworkGraph)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceA)},
			},
		},
		"Test Unrestricted NetworkGraph ReadWrite (override already fully readable), checking read on NetworkGraph for cluster1, then namespaceB": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceNetworkGraph)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceB)},
			},
			wrapperResource: resourceNetworkGraph,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceNetworkGraph)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceB)},
			},
		},
		"Test Unrestricted NetworkGraph ReadWrite (override already fully readable), checking write on NetworkGraph for cluster1, then namespaceB": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceNetworkGraph)},
				{allowed: false, key: sac.ClusterScopeKey(cluster1)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceB)},
			},
			wrapperResource: resourceNetworkGraph,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceNetworkGraph)},
				{allowed: true, key: sac.ClusterScopeKey(cluster1)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceB)},
			},
		},
		"Test Unrestricted NetworkGraph ReadWrite (override already fully readable), checking read on NetworkGraph for cluster2, then namespaceB": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceNetworkGraph)},
				{allowed: true, key: sac.ClusterScopeKey(cluster2)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceB)},
			},
			wrapperResource: resourceNetworkGraph,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceNetworkGraph)},
				{allowed: true, key: sac.ClusterScopeKey(cluster2)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceB)},
			},
		},
		"Test Unrestricted NetworkGraph ReadWrite (override already fully readable), checking write on NetworkGraph for cluster2, then namespaceB": {
			baseSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: false, key: sac.ResourceScopeKey(resourceNetworkGraph)},
				{allowed: false, key: sac.ClusterScopeKey(cluster2)},
				{allowed: false, key: sac.NamespaceScopeKey(namespaceB)},
			},
			wrapperResource: resourceNetworkGraph,
			wrappedSCCDrillDown: []drillDown{
				{allowed: false, key: sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)},
				{allowed: true, key: sac.ResourceScopeKey(resourceNetworkGraph)},
				{allowed: true, key: sac.ClusterScopeKey(cluster2)},
				{allowed: true, key: sac.NamespaceScopeKey(namespaceB)},
			},
		},
	}

	baseTestCtx := sac.WithGlobalAccessScopeChecker(context.Background(), getSampleTestScopeCheckerCore(t))
	for testName, tc := range testCases {
		t.Run(testName, func(it *testing.T) {
			// Validate access drill down on base SCC
			checker := sac.GlobalAccessScopeChecker(baseTestCtx)
			assert.False(it, checker.IsAllowed())
			for _, down := range tc.baseSCCDrillDown {
				fmt.Println("Drilling down to", down.key.String())
				checker = checker.SubScopeChecker(down.key)
				assert.Equal(it, down.allowed, checker.IsAllowed())
			}
			resourceMD, found := resources.MetadataForResource(tc.wrapperResource)
			assert.True(it, found)
			wrappedCtx := sac.WithUnrestrictedResourceReadWrite(baseTestCtx, resourceMD)
			wrappedChecker := sac.GlobalAccessScopeChecker(wrappedCtx)
			assert.False(it, wrappedChecker.IsAllowed())
			for _, down := range tc.wrappedSCCDrillDown {
				fmt.Println("Wrapped - Drilling down to", down.key.String())
				wrappedChecker = wrappedChecker.SubScopeChecker(down.key)
				assert.Equal(it, down.allowed, wrappedChecker.IsAllowed())
			}
		})
	}
}

func TestUnrestrictedResourceReadEffectiveAccessScope(t *testing.T) {
	testCases := map[string]struct {
		baseSCCAccessScope    effectiveaccessscope.ScopeTreeCompacted
		wrapperResource       permissions.Resource
		wrappedSCCAccessScope effectiveaccessscope.ScopeTreeCompacted
		testedResourceAccess  permissions.ResourceWithAccess
	}{
		"Test Unrestricted Image Read, checking read on Cluster": {
			baseSCCAccessScope:    sampleCompactTreeClusterRead,
			wrapperResource:       resourceImage,
			wrappedSCCAccessScope: sampleCompactTreeClusterRead,
			testedResourceAccess:  permissions.View(resources.Cluster),
		},
		"Test Unrestricted Image Read, checking read on Deployment": {
			baseSCCAccessScope:    sampleCompactTreeDeploymentRead,
			wrapperResource:       resourceImage,
			wrappedSCCAccessScope: sampleCompactTreeDeploymentRead,
			testedResourceAccess:  permissions.View(resources.Deployment),
		},
		"Test Unrestricted Image Read, checking read on Image": {
			baseSCCAccessScope:    effectiveaccessscope.DenyAllEffectiveAccessScope().Compactify(),
			wrapperResource:       resourceImage,
			wrappedSCCAccessScope: effectiveaccessscope.UnrestrictedEffectiveAccessScope().Compactify(),
			testedResourceAccess:  permissions.View(resources.Image),
		},
		"Test Unrestricted Image Read, checking write on Image": {
			baseSCCAccessScope:    effectiveaccessscope.DenyAllEffectiveAccessScope().Compactify(),
			wrapperResource:       resourceImage,
			wrappedSCCAccessScope: effectiveaccessscope.DenyAllEffectiveAccessScope().Compactify(),
			testedResourceAccess:  permissions.Modify(resources.Image),
		},
		"Test Unrestricted Deployment Read, checking read on Deployment": {
			baseSCCAccessScope:    sampleCompactTreeDeploymentRead,
			wrapperResource:       resourceDeployment,
			wrappedSCCAccessScope: effectiveaccessscope.UnrestrictedEffectiveAccessScope().Compactify(),
			testedResourceAccess:  permissions.View(resources.Deployment),
		},
		"Test Unrestricted Deployment Read, checking write on Deployment": {
			baseSCCAccessScope:    effectiveaccessscope.DenyAllEffectiveAccessScope().Compactify(),
			wrapperResource:       resourceDeployment,
			wrappedSCCAccessScope: effectiveaccessscope.DenyAllEffectiveAccessScope().Compactify(),
			testedResourceAccess:  permissions.Modify(resources.Deployment),
		},
		"Test Unrestricted NetworkGraph Read, checking read on NetworkGraph": {
			baseSCCAccessScope:    effectiveaccessscope.UnrestrictedEffectiveAccessScope().Compactify(),
			wrapperResource:       resourceNetworkGraph,
			wrappedSCCAccessScope: effectiveaccessscope.UnrestrictedEffectiveAccessScope().Compactify(),
			testedResourceAccess:  permissions.View(resources.NetworkGraph),
		},
		"Test Unrestricted NetworkGraph Read, checking write on NetworkGraph": {
			baseSCCAccessScope:    effectiveaccessscope.DenyAllEffectiveAccessScope().Compactify(),
			wrapperResource:       resourceNetworkGraph,
			wrappedSCCAccessScope: effectiveaccessscope.DenyAllEffectiveAccessScope().Compactify(),
			testedResourceAccess:  permissions.Modify(resources.NetworkGraph),
		},
		"Test Unrestricted NetworkPolicy Read, checking read on NetworkPolicy": {
			baseSCCAccessScope:    sampleCompactTreeNetworkPolicyRead,
			wrapperResource:       resourceNetworkPolicy,
			wrappedSCCAccessScope: effectiveaccessscope.UnrestrictedEffectiveAccessScope().Compactify(),
			testedResourceAccess:  permissions.View(resources.NetworkPolicy),
		},
		"Test Unrestricted NetworkPolicy Read, checking write on NetworkPolicy": {
			baseSCCAccessScope:    sampleCompactTreeNetworkPolicyWrite,
			wrapperResource:       resourceNetworkPolicy,
			wrappedSCCAccessScope: sampleCompactTreeNetworkPolicyWrite,
			testedResourceAccess:  permissions.Modify(resources.NetworkPolicy),
		},
	}

	baseTestCtx := sac.WithGlobalAccessScopeChecker(context.Background(), getSampleTestScopeCheckerCore(t))
	for testName, tc := range testCases {
		t.Run(testName, func(it *testing.T) {
			baseChecker := sac.GlobalAccessScopeChecker(baseTestCtx)
			baseEffectiveScope, err := baseChecker.EffectiveAccessScope(tc.testedResourceAccess)
			assert.NoError(it, err)
			assert.Equal(it, tc.baseSCCAccessScope, baseEffectiveScope.Compactify())
			resourceMD, found := resources.MetadataForResource(tc.wrapperResource)
			assert.True(it, found)
			wrappedCheckerCtx := sac.WithUnrestrictedResourceRead(baseTestCtx, resourceMD)
			wrappedChecker := sac.GlobalAccessScopeChecker(wrappedCheckerCtx)
			wrappedEffectiveScope, err := wrappedChecker.EffectiveAccessScope(tc.testedResourceAccess)
			assert.NoError(it, err)
			assert.Equal(it, tc.wrappedSCCAccessScope, wrappedEffectiveScope.Compactify())
		})
	}
}

func TestUnrestrictedResourceReadWriteEffectiveAccessScope(t *testing.T) {
	testCases := map[string]struct {
		baseSCCAccessScope    effectiveaccessscope.ScopeTreeCompacted
		wrapperResource       permissions.Resource
		wrappedSCCAccessScope effectiveaccessscope.ScopeTreeCompacted
		testedResourceAccess  permissions.ResourceWithAccess
	}{
		"Test Unrestricted Image Read, checking read on Cluster": {
			baseSCCAccessScope:    sampleCompactTreeClusterRead,
			wrapperResource:       resourceImage,
			wrappedSCCAccessScope: sampleCompactTreeClusterRead,
			testedResourceAccess:  permissions.View(resources.Cluster),
		},
		"Test Unrestricted Image Read, checking read on Deployment": {
			baseSCCAccessScope:    sampleCompactTreeDeploymentRead,
			wrapperResource:       resourceImage,
			wrappedSCCAccessScope: sampleCompactTreeDeploymentRead,
			testedResourceAccess:  permissions.View(resources.Deployment),
		},
		"Test Unrestricted Image Read, checking read on Image": {
			baseSCCAccessScope:    effectiveaccessscope.DenyAllEffectiveAccessScope().Compactify(),
			wrapperResource:       resourceImage,
			wrappedSCCAccessScope: effectiveaccessscope.UnrestrictedEffectiveAccessScope().Compactify(),
			testedResourceAccess:  permissions.View(resources.Image),
		},
		"Test Unrestricted Image Read, checking write on Image": {
			baseSCCAccessScope:    effectiveaccessscope.DenyAllEffectiveAccessScope().Compactify(),
			wrapperResource:       resourceImage,
			wrappedSCCAccessScope: effectiveaccessscope.UnrestrictedEffectiveAccessScope().Compactify(),
			testedResourceAccess:  permissions.Modify(resources.Image),
		},
		"Test Unrestricted Deployment Read, checking read on Deployment": {
			baseSCCAccessScope:    sampleCompactTreeDeploymentRead,
			wrapperResource:       resourceDeployment,
			wrappedSCCAccessScope: effectiveaccessscope.UnrestrictedEffectiveAccessScope().Compactify(),
			testedResourceAccess:  permissions.View(resources.Deployment),
		},
		"Test Unrestricted Deployment Read, checking write on Deployment": {
			baseSCCAccessScope:    effectiveaccessscope.DenyAllEffectiveAccessScope().Compactify(),
			wrapperResource:       resourceDeployment,
			wrappedSCCAccessScope: effectiveaccessscope.UnrestrictedEffectiveAccessScope().Compactify(),
			testedResourceAccess:  permissions.Modify(resources.Deployment),
		},
		"Test Unrestricted NetworkGraph Read, checking read on NetworkGraph": {
			baseSCCAccessScope:    effectiveaccessscope.UnrestrictedEffectiveAccessScope().Compactify(),
			wrapperResource:       resourceNetworkGraph,
			wrappedSCCAccessScope: effectiveaccessscope.UnrestrictedEffectiveAccessScope().Compactify(),
			testedResourceAccess:  permissions.View(resources.NetworkGraph),
		},
		"Test Unrestricted NetworkGraph Read, checking write on NetworkGraph": {
			baseSCCAccessScope:    effectiveaccessscope.DenyAllEffectiveAccessScope().Compactify(),
			wrapperResource:       resourceNetworkGraph,
			wrappedSCCAccessScope: effectiveaccessscope.UnrestrictedEffectiveAccessScope().Compactify(),
			testedResourceAccess:  permissions.Modify(resources.NetworkGraph),
		},
		"Test Unrestricted NetworkPolicy Read, checking read on NetworkPolicy": {
			baseSCCAccessScope:    sampleCompactTreeNetworkPolicyRead,
			wrapperResource:       resourceNetworkPolicy,
			wrappedSCCAccessScope: effectiveaccessscope.UnrestrictedEffectiveAccessScope().Compactify(),
			testedResourceAccess:  permissions.View(resources.NetworkPolicy),
		},
		"Test Unrestricted NetworkPolicy Read, checking write on NetworkPolicy": {
			baseSCCAccessScope:    sampleCompactTreeNetworkPolicyWrite,
			wrapperResource:       resourceNetworkPolicy,
			wrappedSCCAccessScope: effectiveaccessscope.UnrestrictedEffectiveAccessScope().Compactify(),
			testedResourceAccess:  permissions.Modify(resources.NetworkPolicy),
		},
	}

	baseTestCtx := sac.WithGlobalAccessScopeChecker(context.Background(), getSampleTestScopeCheckerCore(t))
	for testName, tc := range testCases {
		t.Run(testName, func(it *testing.T) {
			baseChecker := sac.GlobalAccessScopeChecker(baseTestCtx)
			baseEffectiveScope, err := baseChecker.EffectiveAccessScope(tc.testedResourceAccess)
			assert.NoError(it, err)
			assert.Equal(it, tc.baseSCCAccessScope, baseEffectiveScope.Compactify())
			resourceMD, found := resources.MetadataForResource(tc.wrapperResource)
			assert.True(it, found)
			wrappedCheckerCtx := sac.WithUnrestrictedResourceReadWrite(baseTestCtx, resourceMD)
			wrappedChecker := sac.GlobalAccessScopeChecker(wrappedCheckerCtx)
			wrappedEffectiveScope, err := wrappedChecker.EffectiveAccessScope(tc.testedResourceAccess)
			assert.NoError(it, err)
			assert.Equal(it, tc.wrappedSCCAccessScope, wrappedEffectiveScope.Compactify())
		})
	}
}
