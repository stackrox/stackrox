package authorizer

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	rolePkg "github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScopeCacheSharedAcrossResourcesAndVerbs(t *testing.T) {
	var queries atomic.Int32
	checker := newScopeChecker(t.Context(), []permissions.ResolvedRole{role(allResourcesEdit, withAccessTo1Cluster())}, nil,
		func(*storage.SimpleAccessScope_Rules) (*effectiveaccessscope.ScopeTree, error) {
			queries.Add(1)
			return effectiveaccessscope.FromClustersAndNamespacesMap([]string{firstClusterID}, nil), nil
		})
	require.Zero(t, queries.Load(), "creating the checker must not read scope metadata")
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			for _, resource := range []permissions.ResourceMetadata{resources.Deployment, resources.Secret} {
				for _, access := range []storage.Access{storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS} {
					assert.True(t, checker.SubScopeChecker(sac.AccessModeScopeKey(access)).
						SubScopeChecker(sac.ResourceScopeKey(resource.GetResource())).
						SubScopeChecker(sac.ClusterScopeKey(firstClusterID)).
						SubScopeChecker(sac.NamespaceScopeKey(firstNamespaceName)).Allowed())
				}
			}
		})
	}
	wg.Wait()
	require.EqualValues(t, 1, queries.Load())
}

func TestScopeCacheSpecialScopesSkipSQL(t *testing.T) {
	for name, tc := range map[string]struct {
		scope   *storage.SimpleAccessScope
		allowed bool
	}{
		"nil":         {},
		"exclude all": {scope: rolePkg.AccessScopeExcludeAll},
		"include all": {scope: rolePkg.AccessScopeIncludeAll, allowed: true},
	} {
		t.Run(name, func(t *testing.T) {
			checker := newScopeChecker(t.Context(), []permissions.ResolvedRole{role(allResourcesView, tc.scope)}, nil,
				func(*storage.SimpleAccessScope_Rules) (*effectiveaccessscope.ScopeTree, error) {
					t.Fatal("special scopes must not query metadata")
					return nil, nil
				})
			assert.Equal(t, tc.allowed, checker.SubScopeChecker(sac.AccessModeScopeKey(storage.Access_READ_ACCESS)).
				SubScopeChecker(sac.ResourceScopeKey(resources.Deployment.GetResource())).
				SubScopeChecker(sac.ClusterScopeKey(firstClusterID)).
				SubScopeChecker(sac.NamespaceScopeKey(firstNamespaceName)).Allowed())
		})
	}
}

func TestScopeCacheKeepsReadAndWriteGrantsPairedWithRoles(t *testing.T) {
	readScope := &storage.SimpleAccessScope{Id: "read", Rules: &storage.SimpleAccessScope_Rules{IncludedClusterIds: []string{firstClusterID}}}
	writeScope := &storage.SimpleAccessScope{Id: "write", Rules: &storage.SimpleAccessScope_Rules{IncludedClusterIds: []string{secondClusterID}}}
	var queries int
	checker := newScopeChecker(t.Context(), []permissions.ResolvedRole{
		role(map[string]storage.Access{resources.Deployment.String(): storage.Access_READ_ACCESS}, readScope),
		role(map[string]storage.Access{resources.Deployment.String(): storage.Access_READ_WRITE_ACCESS}, writeScope),
	}, nil, func(rules *storage.SimpleAccessScope_Rules) (*effectiveaccessscope.ScopeTree, error) {
		queries++
		clusterID := rules.GetIncludedClusterIds()[0]
		return effectiveaccessscope.FromMatchingScopes([]effectiveaccessscope.Cluster{&storage.Cluster{Id: clusterID, Name: clusterID}}, nil,
			[]effectiveaccessscope.MatchedNamespaceGroup{{ClusterName: clusterID, Complete: true,
				Namespaces:         []effectiveaccessscope.MatchedNamespace{{Name: firstNamespaceName}, {Name: "other"}, {Name: "third"}},
				ExcludedNamespaces: []effectiveaccessscope.MatchedNamespace{{Name: "excluded"}},
			}}), nil
	})
	allowed := func(access storage.Access, resource permissions.ResourceMetadata, clusterID string) bool {
		return checker.SubScopeChecker(sac.AccessModeScopeKey(access)).SubScopeChecker(sac.ResourceScopeKey(resource.GetResource())).
			SubScopeChecker(sac.ClusterScopeKey(clusterID)).SubScopeChecker(sac.NamespaceScopeKey(firstNamespaceName)).Allowed()
	}
	require.True(t, allowed(storage.Access_READ_ACCESS, resources.Deployment, firstClusterID))
	require.False(t, allowed(storage.Access_READ_WRITE_ACCESS, resources.Deployment, firstClusterID))
	require.True(t, allowed(storage.Access_READ_WRITE_ACCESS, resources.Deployment, secondClusterID))
	require.False(t, allowed(storage.Access_READ_ACCESS, resources.Secret, secondClusterID))
	require.Equal(t, 2, queries)
}

func TestScopeCacheRetainsQueryErrors(t *testing.T) {
	queryError := errors.New("scope query failed")
	var queries int
	checker := newScopeChecker(t.Context(), []permissions.ResolvedRole{role(allResourcesView, withAccessTo1Cluster())}, nil,
		func(*storage.SimpleAccessScope_Rules) (*effectiveaccessscope.ScopeTree, error) {
			queries++
			return nil, queryError
		})
	for range 3 {
		tree, err := checker.EffectiveAccessScope(permissions.ResourceWithAccess{
			Resource: resources.Deployment, Access: storage.Access_READ_ACCESS,
		})
		require.ErrorIs(t, err, queryError)
		require.Nil(t, tree)
	}
	require.Equal(t, 1, queries)
}

func TestScopeCacheSlowLookupDoesNotBlockOtherScopes(t *testing.T) {
	for name, cached := range map[string]bool{"new scope": false, "cached scope": true} {
		t.Run(name, func(t *testing.T) {
			started := make(chan struct{})
			release := make(chan struct{})
			var wg sync.WaitGroup
			defer wg.Wait()
			defer close(release)
			slowScope := &storage.SimpleAccessScope{Id: "slow", Rules: &storage.SimpleAccessScope_Rules{IncludedClusterIds: []string{firstClusterID}}}
			fastScope := &storage.SimpleAccessScope{Id: "fast", Rules: &storage.SimpleAccessScope_Rules{IncludedClusterIds: []string{secondClusterID}}}
			var fastQueries atomic.Int32
			checker := newScopeChecker(t.Context(), []permissions.ResolvedRole{
				role(map[string]storage.Access{resources.Deployment.String(): storage.Access_READ_ACCESS}, slowScope),
				role(map[string]storage.Access{resources.Secret.String(): storage.Access_READ_ACCESS}, fastScope),
			}, nil, func(rules *storage.SimpleAccessScope_Rules) (*effectiveaccessscope.ScopeTree, error) {
				if rules == slowScope.GetRules() {
					close(started)
					<-release
				} else {
					fastQueries.Add(1)
				}
				return effectiveaccessscope.FromClustersAndNamespacesMap(rules.GetIncludedClusterIds(), nil), nil
			})
			fastLookup := func() {
				tree, err := checker.EffectiveAccessScope(permissions.ResourceWithAccess{Resource: resources.Secret, Access: storage.Access_READ_ACCESS})
				assert.NoError(t, err)
				assert.NotNil(t, tree)
			}
			if cached {
				fastLookup()
			}
			wg.Go(func() {
				_, err := checker.EffectiveAccessScope(permissions.ResourceWithAccess{Resource: resources.Deployment, Access: storage.Access_READ_ACCESS})
				assert.NoError(t, err)
			})
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("slow lookup did not start")
			}
			done := make(chan struct{})
			wg.Go(func() {
				defer close(done)
				fastLookup()
			})
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("another scope was blocked by the slow lookup")
			}
			require.EqualValues(t, 1, fastQueries.Load())
		})
	}
}

func TestScopeCacheWaitCanBeCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	started := make(chan struct{})
	release := make(chan struct{})
	var wg sync.WaitGroup
	defer wg.Wait()
	defer close(release)
	checker := newScopeChecker(ctx, []permissions.ResolvedRole{role(allResourcesView, withAccessTo1Cluster())}, nil,
		func(*storage.SimpleAccessScope_Rules) (*effectiveaccessscope.ScopeTree, error) {
			close(started)
			<-release
			return nil, ctx.Err()
		})
	resource := permissions.ResourceWithAccess{Resource: resources.Deployment, Access: storage.Access_READ_ACCESS}
	wg.Go(func() {
		_, err := checker.EffectiveAccessScope(resource)
		assert.ErrorIs(t, err, context.Canceled)
	})
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("scope lookup did not start")
	}
	cancel()
	done := make(chan struct{})
	wg.Go(func() {
		defer close(done)
		tree, err := checker.EffectiveAccessScope(resource)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Nil(t, tree)
	})
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scope waiter did not observe cancellation")
	}
}
