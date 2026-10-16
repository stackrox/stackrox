//go:build sql_integration

package authorizer

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	clusterPostgres "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	deploymentStore "github.com/stackrox/rox/central/deployment/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sac/observe"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scopeQueryFixtures(t testing.TB, db *pgtest.TestPostgres) ([]effectiveaccessscope.Cluster, []effectiveaccessscope.Namespace) {
	ctx := sac.WithAllAccess(context.Background())
	clusters := []*storage.Cluster{
		{Id: uuid.NewV4().String(), Name: "Earth"},
		{Id: uuid.NewV4().String(), Name: "Arrakis", Labels: map[string]string{"focus": "melange", effectiveaccessscope.ClusterNameLabel: "Earth"}},
		{Id: uuid.NewV4().String(), Name: "Other", Labels: map[string]string{"focus": ""}},
	}
	require.NoError(t, clusterPostgres.New(db).UpsertMany(ctx, clusters))
	namespaces := []*storage.NamespaceMetadata{
		{Id: uuid.NewV4().String(), Name: "glorp", ClusterId: clusters[0].GetId(), ClusterName: "Earth", Labels: map[string]string{"focus": "melange", "region": "NA"}},
		{Id: uuid.NewV4().String(), Name: "gleep", ClusterId: clusters[0].GetId(), ClusterName: "Earth"},
		{Id: uuid.NewV4().String(), Name: "empty", ClusterId: clusters[0].GetId(), ClusterName: "Earth", Labels: map[string]string{"focus": ""}},
		{Id: uuid.NewV4().String(), Name: "glorp", ClusterId: clusters[1].GetId(), ClusterName: "Arrakis", Labels: map[string]string{"focus": "research", "region": "EU", effectiveaccessscope.NamespaceNameLabel: "spoof"}},
		{Id: uuid.NewV4().String(), Name: "orphan", ClusterId: uuid.NewV4().String(), ClusterName: "Deleted", Labels: map[string]string{"focus": "melange"}},
		{Id: uuid.NewV4().String(), Name: "inconsistent", ClusterId: clusters[0].GetId(), ClusterName: "Arrakis", Labels: map[string]string{"focus": "melange"}},
		{Id: uuid.NewV4().String(), Name: "glorp", ClusterId: clusters[0].GetId(), ClusterName: "Earth", Labels: map[string]string{"focus": "research"}},
		{Id: uuid.NewV4().String(), Name: "glorp' OR TRUE --", ClusterId: clusters[0].GetId(), ClusterName: "Earth"},
		{Id: uuid.NewV4().String(), Name: "", ClusterId: clusters[0].GetId(), ClusterName: "Earth", Labels: map[string]string{"focus": "melange"}},
		{Id: uuid.NewV4().String(), Name: "café::'namespace", ClusterId: clusters[0].GetId(), ClusterName: "Earth", Labels: map[string]string{"focus": "melange"}},
	}
	rows := make([][]any, 0, len(namespaces))
	for _, namespace := range namespaces {
		serialized, err := namespace.MarshalVT()
		require.NoError(t, err)
		rows = append(rows, []any{namespace.GetId(), namespace.GetName(), namespace.GetClusterId(), namespace.GetClusterName(), namespace.GetLabels(), serialized})
	}
	connection, err := db.Acquire(ctx)
	require.NoError(t, err)
	_, err = connection.CopyFrom(ctx, pgx.Identifier{"namespaces"},
		[]string{"id", "name", "clusterid", "clustername", "labels", "serialized"}, pgx.CopyFromRows(rows))
	connection.Release()
	require.NoError(t, err)
	// Exercise SQL NULL as well as JSON null and empty maps for missing labels.
	_, err = db.Exec(ctx, "UPDATE clusters SET labels = NULL WHERE id = $1", clusters[0].GetId())
	require.NoError(t, err)
	_, err = db.Exec(ctx, "UPDATE namespaces SET labels = NULL WHERE id = $1", namespaces[1].GetId())
	require.NoError(t, err)
	resultClusters := make([]effectiveaccessscope.Cluster, 0, len(clusters))
	for _, cluster := range clusters {
		resultClusters = append(resultClusters, cluster)
	}
	resultNamespaces := make([]effectiveaccessscope.Namespace, 0, len(namespaces))
	for _, namespace := range namespaces {
		resultNamespaces = append(resultNamespaces, namespace)
	}
	return resultClusters, resultNamespaces
}

func scopeRequirement(key string, op storage.SetBasedLabelSelector_Operator, values ...string) *storage.SetBasedLabelSelector_Requirement {
	return &storage.SetBasedLabelSelector_Requirement{Key: key, Op: op, Values: values}
}

func TestSQLScopeMatchesInventoryEvaluator(t *testing.T) {
	db := pgtest.ForT(t)
	clusters, namespaces := scopeQueryFixtures(t, db)
	clusterID := clusters[0].GetId()
	tests := map[string]*storage.SimpleAccessScope_Rules{
		"nil":                      nil,
		"empty":                    {},
		"cluster by ID":            {IncludedClusterIds: []string{clusterID}},
		"cluster by name":          {IncludedClusters: []string{"Arrakis"}},
		"unknown cluster":          {IncludedClusterIds: []string{uuid.NewV4().String()}},
		"malformed cluster ID":     {IncludedClusterIds: []string{"not-a-uuid"}, IncludedClusters: []string{"Earth"}},
		"noncanonical ID":          {IncludedClusterIds: []string{strings.ToUpper(clusterID)}},
		"namespace by ID":          {IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{{ClusterId: clusterID, NamespaceName: "glorp"}}},
		"namespace by name":        {IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{{ClusterName: "Earth", NamespaceName: "glorp"}}},
		"ID precedes name":         {IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{{ClusterId: clusterID, ClusterName: "Arrakis", NamespaceName: "glorp"}}},
		"unknown ID precedes name": {IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{{ClusterId: uuid.NewV4().String(), ClusterName: "Earth", NamespaceName: "glorp"}}},
		"invalid ID precedes name": {IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{{ClusterId: "bad", ClusterName: "Earth", NamespaceName: "glorp"}}},
		"missing namespace fields": {IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{nil, {}, {ClusterId: clusterID}, {NamespaceName: "glorp"}}},
		"parameterized names":      {IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{{ClusterName: "Earth", NamespaceName: "glorp' OR TRUE --"}}},
		"empty label selectors":    {ClusterLabelSelectors: []*storage.SetBasedLabelSelector{nil, {}}, NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{nil, {}}},
	}
	for field, keys := range map[string][]string{
		"cluster":   {"focus", "absent", effectiveaccessscope.ClusterNameLabel},
		"namespace": {"focus", "absent", effectiveaccessscope.NamespaceNameLabel},
	} {
		for _, key := range keys {
			for _, op := range []storage.SetBasedLabelSelector_Operator{
				storage.SetBasedLabelSelector_IN, storage.SetBasedLabelSelector_NOT_IN,
				storage.SetBasedLabelSelector_EXISTS, storage.SetBasedLabelSelector_NOT_EXISTS,
			} {
				var values []string
				if op == storage.SetBasedLabelSelector_IN || op == storage.SetBasedLabelSelector_NOT_IN {
					values = []string{"melange", "Earth", ""}
				}
				rules := &storage.SimpleAccessScope_Rules{}
				selector := &storage.SetBasedLabelSelector{Requirements: []*storage.SetBasedLabelSelector_Requirement{scopeRequirement(key, op, values...)}}
				if field == "cluster" {
					rules.ClusterLabelSelectors = []*storage.SetBasedLabelSelector{selector}
				} else {
					rules.NamespaceLabelSelectors = []*storage.SetBasedLabelSelector{selector}
				}
				tests[fmt.Sprintf("%s %s %s", field, key, op)] = rules
			}
		}
	}
	tests["requirements AND selectors OR"] = &storage.SimpleAccessScope_Rules{NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{
		{Requirements: []*storage.SetBasedLabelSelector_Requirement{scopeRequirement("focus", storage.SetBasedLabelSelector_IN, "melange"), scopeRequirement("region", storage.SetBasedLabelSelector_IN, "NA")}},
		{Requirements: []*storage.SetBasedLabelSelector_Requirement{scopeRequirement("region", storage.SetBasedLabelSelector_IN, "EU")}},
	}}
	tests["IN and NOT_IN conjunction"] = &storage.SimpleAccessScope_Rules{NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{{
		Requirements: []*storage.SetBasedLabelSelector_Requirement{scopeRequirement("focus", storage.SetBasedLabelSelector_IN, "melange"), scopeRequirement("region", storage.SetBasedLabelSelector_NOT_IN, "EU")},
	}}}
	tests["positive and negative alternatives"] = &storage.SimpleAccessScope_Rules{NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{
		{Requirements: []*storage.SetBasedLabelSelector_Requirement{scopeRequirement("focus", storage.SetBasedLabelSelector_IN, "melange")}},
		{Requirements: []*storage.SetBasedLabelSelector_Requirement{scopeRequirement("region", storage.SetBasedLabelSelector_NOT_IN, "NA")}},
	}}
	tests["negative grant and explicit cluster"] = &storage.SimpleAccessScope_Rules{
		IncludedClusterIds: []string{clusterID},
		NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{{Requirements: []*storage.SetBasedLabelSelector_Requirement{
			scopeRequirement("focus", storage.SetBasedLabelSelector_NOT_IN, "melange"),
		}}},
	}
	tests["negative requirements conjunction"] = &storage.SimpleAccessScope_Rules{NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{{
		Requirements: []*storage.SetBasedLabelSelector_Requirement{scopeRequirement("focus", storage.SetBasedLabelSelector_NOT_IN, "melange"), scopeRequirement("region", storage.SetBasedLabelSelector_NOT_EXISTS)},
	}}}
	tests["negative grant and explicit namespace"] = &storage.SimpleAccessScope_Rules{
		IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{{ClusterId: clusterID, NamespaceName: "glorp"}},
		NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{{Requirements: []*storage.SetBasedLabelSelector_Requirement{
			scopeRequirement("focus", storage.SetBasedLabelSelector_NOT_IN, "melange", "research"),
		}}},
	}
	tests["full clusters union namespaces"] = &storage.SimpleAccessScope_Rules{
		IncludedClusterIds:      []string{clusterID},
		NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{{Requirements: []*storage.SetBasedLabelSelector_Requirement{scopeRequirement("focus", storage.SetBasedLabelSelector_EXISTS)}}},
	}
	for _, requirement := range []*storage.SetBasedLabelSelector_Requirement{
		nil, scopeRequirement("invalid key", storage.SetBasedLabelSelector_EXISTS),
		scopeRequirement("focus", storage.SetBasedLabelSelector_UNKNOWN),
		scopeRequirement("focus", storage.SetBasedLabelSelector_IN),
		scopeRequirement("focus", storage.SetBasedLabelSelector_EXISTS, "unexpected"),
		scopeRequirement(effectiveaccessscope.NamespaceNameLabel, storage.SetBasedLabelSelector_IN, "Earth::glorp"),
	} {
		tests[fmt.Sprintf("invalid requirement %v", requirement)] = &storage.SimpleAccessScope_Rules{
			IncludedClusterIds:      []string{clusterID},
			NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{{Requirements: []*storage.SetBasedLabelSelector_Requirement{requirement}}},
		}
	}
	for name, rules := range tests {
		t.Run(name, func(t *testing.T) {
			expected, expectedErr := effectiveaccessscope.ComputeEffectiveAccessScope(rules, clusters, namespaces, v1.ComputeEffectiveAccessScopeRequest_MINIMAL)
			actual, err := computeSQLScope(context.Background(), db, rules)
			if expectedErr != nil {
				require.Error(t, err)
				require.Nil(t, actual)
				return
			}
			require.NoError(t, err)
			require.Equal(t, expected.State, actual.State)
			require.Equal(t, expected.Compactify(), actual.Compactify())
			for _, cluster := range clusters {
				assert.Equal(t, effectiveAccessScopeAllows(expected, resources.Deployment, cluster.GetId(), "unknown"),
					effectiveAccessScopeAllows(actual, resources.Deployment, cluster.GetId(), "unknown"))
				for _, namespace := range namespaces {
					for _, resource := range []permissions.ResourceMetadata{resources.Cluster, resources.Deployment} {
						assert.Equal(t, effectiveAccessScopeAllows(expected, resource, cluster.GetId(), namespace.GetName()),
							effectiveAccessScopeAllows(actual, resource, cluster.GetId(), namespace.GetName()))
					}
				}
			}
		})
	}
}

func TestSQLScopeExclusionSnapshot(t *testing.T) {
	db := pgtest.ForT(t)
	clusters, namespaces := scopeQueryFixtures(t, db)
	rules := &storage.SimpleAccessScope_Rules{NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{{
		Requirements: []*storage.SetBasedLabelSelector_Requirement{scopeRequirement("absent", storage.SetBasedLabelSelector_NOT_EXISTS)},
	}}}
	ctx := context.Background()
	before, err := computeSQLScope(ctx, db, rules)
	require.NoError(t, err)
	earth := before.GetClusterByID(clusters[0].GetId())
	require.NotNil(t, earth)
	require.Empty(t, earth.Namespaces) // All known names are implicitly included.
	require.True(t, earth.NamespaceAllowed("gleep"))
	require.False(t, earth.NamespaceAllowed("future"))
	_, err = db.Exec(ctx, "UPDATE namespaces SET name = $1 WHERE id = $2", "future", namespaces[1].GetId())
	require.NoError(t, err)
	after, err := computeSQLScope(ctx, db, rules)
	require.NoError(t, err)
	assert.True(t, earth.NamespaceAllowed("gleep"))
	assert.False(t, earth.NamespaceAllowed("future"))
	assert.False(t, after.GetClusterByID(clusters[0].GetId()).NamespaceAllowed("gleep"))
	assert.True(t, after.GetClusterByID(clusters[0].GetId()).NamespaceAllowed("future"))

	duplicateRules := &storage.SimpleAccessScope_Rules{NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{{
		Requirements: []*storage.SetBasedLabelSelector_Requirement{scopeRequirement("focus", storage.SetBasedLabelSelector_NOT_IN, "research")},
	}}}
	duplicate, err := computeSQLScope(ctx, db, duplicateRules)
	require.NoError(t, err)
	assert.True(t, duplicate.GetClusterByID(clusters[0].GetId()).NamespaceAllowed("glorp"))
	assert.Equal(t, namespaces[0].GetId(), duplicate.GetClusterByID(clusters[0].GetId()).GetNamespaceByName("glorp").Attributes.ID)
}

// Packing matched IDs/names must preserve arbitrary names and the deterministic
// ID chosen for duplicate recorded namespace names even without SQL sorting.
func TestSQLScopePreservesNamespaceMetadata(t *testing.T) {
	db := pgtest.ForT(t)
	clusters, namespaces := scopeQueryFixtures(t, db)
	rules := &storage.SimpleAccessScope_Rules{NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{{
		Requirements: []*storage.SetBasedLabelSelector_Requirement{scopeRequirement("focus", storage.SetBasedLabelSelector_EXISTS)},
	}}}
	tree, err := computeSQLScope(context.Background(), db, rules)
	require.NoError(t, err)
	earth := tree.GetClusterByID(clusters[0].GetId())
	require.NotNil(t, earth)
	require.Equal(t, min(namespaces[0].GetId(), namespaces[6].GetId()), earth.Namespaces["glorp"].Attributes.ID)
	require.Equal(t, namespaces[8].GetId(), earth.Namespaces[""].Attributes.ID)
	require.Equal(t, namespaces[9].GetId(), earth.Namespaces["café::'namespace"].Attributes.ID)
	require.Equal(t, namespaces[4].GetId(), tree.Clusters["Deleted"].Namespaces["orphan"].Attributes.ID)
	arrakis := tree.GetClusterByID(clusters[1].GetId())
	require.NotNil(t, arrakis)
	require.Equal(t, namespaces[5].GetId(), arrakis.Namespaces["inconsistent"].Attributes.ID)
}

func TestSQLScopeFreshnessAndCancellation(t *testing.T) {
	db := pgtest.ForT(t)
	clusters, _ := scopeQueryFixtures(t, db)
	rules := &storage.SimpleAccessScope_Rules{ClusterLabelSelectors: []*storage.SetBasedLabelSelector{{
		Requirements: []*storage.SetBasedLabelSelector_Requirement{scopeRequirement("focus", storage.SetBasedLabelSelector_IN, "melange")},
	}}}
	ctx := context.Background()
	newChecker := func() sac.ScopeCheckerCore {
		return newScopeChecker([]permissions.ResolvedRole{role(allResourcesView, &storage.SimpleAccessScope{Id: "label-scope", Rules: rules})}, nil,
			func(rules *storage.SimpleAccessScope_Rules) (*effectiveaccessscope.ScopeTree, error) {
				return computeSQLScope(ctx, db, rules)
			})
	}
	resource := permissions.ResourceWithAccess{Resource: resources.Deployment, Access: storage.Access_READ_ACCESS}
	checker := newChecker()
	before, err := checker.EffectiveAccessScope(resource)
	require.NoError(t, err)
	require.True(t, effectiveAccessScopeAllows(before, resources.Deployment, clusters[1].GetId(), "any"))
	_, err = db.Exec(ctx, `UPDATE clusters SET labels = '{"focus":"other"}'::jsonb WHERE id = $1`, clusters[1].GetId())
	require.NoError(t, err)
	cached, err := checker.EffectiveAccessScope(resource)
	require.NoError(t, err)
	require.True(t, effectiveAccessScopeAllows(cached, resources.Deployment, clusters[1].GetId(), "any"))
	after, err := newChecker().EffectiveAccessScope(resource)
	require.NoError(t, err)
	require.False(t, effectiveAccessScopeAllows(after, resources.Deployment, clusters[1].GetId(), "any"))

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	tree, err := computeSQLScope(canceled, db, rules)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, tree)
	db.Close()
	tree, err = computeSQLScope(ctx, db, rules)
	require.Error(t, err)
	require.Nil(t, tree)
}

func TestSQLScopeTracingInventory(t *testing.T) {
	db := pgtest.ForT(t)
	_, _ = scopeQueryFixtures(t, db)
	trace := observe.NewAuthzTrace()
	require.NoError(t, recordScopeInventory(context.Background(), db, trace))
}

func TestSQLScopeFiltersDeploymentReads(t *testing.T) {
	db := pgtest.ForT(t)
	clusters, _ := scopeQueryFixtures(t, db)
	deployments, err := deploymentStore.GetTestPostgresDataStore(t, db)
	require.NoError(t, err)
	ctx := sac.WithAllAccess(context.Background())
	allowed := &storage.Deployment{Id: uuid.NewV4().String(), Name: "allowed", ClusterId: clusters[0].GetId(), ClusterName: "Earth", Namespace: "glorp"}
	denied := &storage.Deployment{Id: uuid.NewV4().String(), Name: "denied", ClusterId: clusters[0].GetId(), ClusterName: "Earth", Namespace: "gleep"}
	otherCluster := &storage.Deployment{Id: uuid.NewV4().String(), Name: "other", ClusterId: clusters[1].GetId(), ClusterName: "Arrakis", Namespace: "glorp"}
	labelMatch := &storage.Deployment{Id: uuid.NewV4().String(), Name: "label", ClusterId: clusters[0].GetId(), ClusterName: "Earth", Namespace: "café::'namespace"}
	unknown := &storage.Deployment{Id: uuid.NewV4().String(), Name: "unknown", ClusterId: clusters[0].GetId(), ClusterName: "Earth", Namespace: "unknown"}
	allDeployments := []*storage.Deployment{allowed, denied, otherCluster, labelMatch, unknown}
	for _, deployment := range allDeployments {
		require.NoError(t, deployments.UpsertDeployment(ctx, deployment))
	}
	labelRules := func(op storage.SetBasedLabelSelector_Operator, value string) *storage.SimpleAccessScope_Rules {
		return &storage.SimpleAccessScope_Rules{NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{{Requirements: []*storage.SetBasedLabelSelector_Requirement{
			scopeRequirement("focus", op, value),
		}}}}
	}
	for name, test := range map[string]struct {
		rules      []*storage.SimpleAccessScope_Rules
		allowedIDs []string
	}{
		"namespace name": {
			rules:      []*storage.SimpleAccessScope_Rules{{IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{{ClusterName: "Earth", NamespaceName: "glorp"}}}},
			allowedIDs: []string{allowed.GetId()},
		},
		"namespace label":                             {rules: []*storage.SimpleAccessScope_Rules{labelRules(storage.SetBasedLabelSelector_IN, "melange")}, allowedIDs: []string{allowed.GetId(), labelMatch.GetId()}},
		"namespace exclusions":                        {rules: []*storage.SimpleAccessScope_Rules{labelRules(storage.SetBasedLabelSelector_NOT_IN, "research")}, allowedIDs: []string{allowed.GetId(), denied.GetId(), labelMatch.GetId()}},
		"namespace exclusions with denied deployment": {rules: []*storage.SimpleAccessScope_Rules{labelRules(storage.SetBasedLabelSelector_NOT_IN, "melange")}, allowedIDs: []string{allowed.GetId(), denied.GetId(), otherCluster.GetId()}},
		"opposing exclusions in different roles": {
			rules:      []*storage.SimpleAccessScope_Rules{labelRules(storage.SetBasedLabelSelector_NOT_IN, "research"), labelRules(storage.SetBasedLabelSelector_NOT_IN, "melange")},
			allowedIDs: []string{allowed.GetId(), denied.GetId(), otherCluster.GetId(), labelMatch.GetId()},
		},
		"inclusion overrides excluded name": {
			rules:      []*storage.SimpleAccessScope_Rules{labelRules(storage.SetBasedLabelSelector_NOT_IN, "research"), {IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{{ClusterName: "Arrakis", NamespaceName: "glorp"}}}},
			allowedIDs: []string{allowed.GetId(), denied.GetId(), otherCluster.GetId(), labelMatch.GetId()},
		},
	} {
		t.Run(name, func(t *testing.T) {
			roles := make([]permissions.ResolvedRole, 0, len(test.rules))
			for i, rules := range test.rules {
				roles = append(roles, role(allResourcesView, &storage.SimpleAccessScope{Id: fmt.Sprintf("scope-%d", i), Rules: rules}))
			}
			checker := newScopeChecker(roles, nil,
				func(rules *storage.SimpleAccessScope_Rules) (*effectiveaccessscope.ScopeTree, error) {
					return computeSQLScope(ctx, db, rules)
				})
			requestCtx := sac.WithGlobalAccessScopeChecker(ctx, checker)
			for _, deployment := range allDeployments {
				_, found, err := deployments.GetDeployment(requestCtx, deployment.GetId())
				require.NoError(t, err)
				assert.Equal(t, slices.Contains(test.allowedIDs, deployment.GetId()), found, deployment.GetName())
			}
			results, err := deployments.SearchRawDeployments(requestCtx, &v1.Query{})
			require.NoError(t, err)
			ids := make([]string, 0, len(results))
			for _, result := range results {
				ids = append(ids, result.GetId())
			}
			assert.ElementsMatch(t, test.allowedIDs, ids)
		})
	}
}
