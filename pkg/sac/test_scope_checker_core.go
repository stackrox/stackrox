package sac

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
)

// TestClusterScope contains cluster-level scope information (cluster included or list of included namespaces
// in the cluster) to build test scope checkers.
type TestClusterScope struct {
	Namespaces []string
	Included   bool
}

// TestResourceScope contains resource-level scope information (resource fully included or list of included clusters
// and associated cluster-level scope information) to build test scope checkers.
type TestResourceScope struct {
	Clusters map[string]*TestClusterScope
	Included bool
}

// TestScopeMap is an abstraction for the scope element hierarchy to build test scope chekers.
type TestScopeMap map[storage.Access]map[permissions.Resource]*TestResourceScope

type testScopeCheckerCore struct {
	scope TestScopeMap
	path  []ScopeKey
}

// TestScopeCheckerCoreFromAccessResourceMap creates a ScopeCheckerCore that allows full access to the input
// (accessMode, Resource) pairs for testing purposes.
func TestScopeCheckerCoreFromAccessResourceMap(_ *testing.T, targetResources []permissions.ResourceWithAccess) ScopeCheckerCore {
	includedResourceScope := &TestResourceScope{
		Included: true,
	}
	core := &testScopeCheckerCore{
		scope: make(TestScopeMap, 0),
	}
	for _, resource := range targetResources {
		access := resource.Access
		if _, accessExists := core.scope[access]; !accessExists {
			core.scope[access] = make(map[permissions.Resource]*TestResourceScope, 0)
		}
		core.scope[access][resource.Resource.GetResource()] = includedResourceScope
	}
	return core
}

// TestScopeCheckerCoreFromFullScopeMap creates a ScopeCheckerCore that allows scoped access to the input
// scope tree for testing purposes.
func TestScopeCheckerCoreFromFullScopeMap(_ *testing.T, targetScope TestScopeMap) ScopeCheckerCore {
	return &testScopeCheckerCore{
		scope: targetScope,
	}
}

func (c *testScopeCheckerCore) EffectiveAccessScope(resource permissions.ResourceWithAccess) (*effectiveaccessscope.ScopeTree, error) {
	resourceMap := c.scope[resource.Access]
	if len(resourceMap) == 0 {
		return effectiveaccessscope.DenyAllEffectiveAccessScope(), nil
	}
	resourceCore := resourceMap[resource.Resource.GetResource()]
	if resourceCore == nil {
		return effectiveaccessscope.DenyAllEffectiveAccessScope(), nil
	}
	if !resourceCore.Included && len(resourceCore.Clusters) == 0 {
		return effectiveaccessscope.DenyAllEffectiveAccessScope(), nil
	}
	if resourceCore.Included {
		return effectiveaccessscope.UnrestrictedEffectiveAccessScope(), nil
	}
	includedClusters := make([]string, 0, len(resourceCore.Clusters))
	includedClusterNamespacePairs := make(map[string][]string, 0)
	for clusterID, clusterScope := range resourceCore.Clusters {
		if clusterScope == nil {
			continue
		}
		if clusterScope.Included {
			includedClusters = append(includedClusters, clusterID)
		}
		for _, namespace := range clusterScope.Namespaces {
			if _, clusterExists := includedClusterNamespacePairs[clusterID]; !clusterExists {
				includedClusterNamespacePairs[clusterID] = make([]string, 0, len(clusterScope.Namespaces))
			}
			includedClusterNamespacePairs[clusterID] = append(includedClusterNamespacePairs[clusterID], namespace)
		}
	}
	return effectiveaccessscope.FromClustersAndNamespacesMap(includedClusters, includedClusterNamespacePairs), nil
}

func (c *testScopeCheckerCore) PerformChecks(_ context.Context) error {
	return nil
}

func (c *testScopeCheckerCore) SubScopeChecker(key ScopeKey) ScopeCheckerCore {
	return &testScopeCheckerCore{
		scope: c.scope,
		path:  append(c.path, key),
	}
}

func (c *testScopeCheckerCore) TryAllowed() TryAllowedResult {
	// Global access is denied, need to drill down.
	if len(c.path) == 0 {
		return Deny
	}
	// Drill down to access level.
	access := c.path[0]
	accessKey, accessOK := access.(AccessModeScopeKey)
	if !accessOK {
		return Deny
	}
	accessMode := storage.Access(accessKey)
	if _, accessAllowed := c.scope[accessMode]; !accessAllowed {
		return Deny
	}
	if len(c.path) == 1 {
		return Deny
	}
	// Drill down to resource level.
	resource := c.path[1]
	resourceKey, resourceOK := resource.(ResourceScopeKey)
	if !resourceOK {
		return Deny
	}
	resourceScope := c.scope[accessMode][permissions.Resource(resourceKey.String())]
	if resourceScope == nil {
		return Deny
	}
	if resourceScope.Included {
		return Allow
	}
	if len(c.path) == 2 {
		return Deny
	}
	// Drill down to cluster level.
	clusterID := c.path[2].String()
	clusterScope := resourceScope.Clusters[clusterID]
	if clusterScope == nil {
		return Deny
	}
	if clusterScope.Included {
		return Allow
	}
	if len(c.path) == 3 {
		return Deny
	}
	// Drill down to namespace level.
	namespace := c.path[3].String()
	namespaceAllowed := false
	for _, allowedNamespace := range clusterScope.Namespaces {
		if namespace == allowedNamespace {
			namespaceAllowed = true
			break
		}
	}
	if namespaceAllowed {
		return Allow
	}
	return Deny
}
