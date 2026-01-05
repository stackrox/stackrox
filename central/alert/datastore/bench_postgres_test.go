//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/stackrox/rox/central/alert/mappings"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Real default policy IDs from pkg/defaults/policies/files/*.json
var defaultPolicyIDs = []struct {
	id       string
	name     string
	severity storage.Severity
}{
	{"2e90874a-3521-44de-85c6-5720f519a701", "Latest tag", storage.Severity_LOW_SEVERITY},
	{"fe9de18b-86db-44d5-a7c4-74173ccffe2e", "Privileged Container", storage.Severity_MEDIUM_SEVERITY},
	{"886c3c94-3a6a-4f2b-82fc-d6bf5a310840", "No CPU request or memory limit specified", storage.Severity_MEDIUM_SEVERITY},
	{"f09f8da1-6111-4ca0-8f49-294a76c65115", "Fixable CVSS >= 7", storage.Severity_HIGH_SEVERITY},
	{"cf80fb33-c7d0-4490-b6f4-e56e1f27b4e4", "Log4Shell: log4j Remote Code Execution vulnerability", storage.Severity_CRITICAL_SEVERITY},
}

func BenchmarkAlertDatabaseOps(b *testing.B) {
	testDB := pgtest.ForT(b)
	ctx := sac.WithAllAccess(context.Background())
	datastore := GetTestPostgresDataStore(b, testDB.DB)

	var ids []string
	// Deployment IDs to distribute alerts across (60/25/15 ratio)
	deploymentIDs := []string{
		fixtureconsts.Deployment1,
		fixtureconsts.Deployment2,
		fixtureconsts.Deployment3,
	}
	sevToCount := make(map[storage.Severity]int)

	// Keep the count low in CI. You can run w/ higher numbers locally.
	totalAlerts := 1000
	for i := 0; i < totalAlerts; i++ {
		id := uuid.NewV4().String()
		ids = append(ids, id)
		a := fixtures.GetAlertWithID(id)

		// Distribute alerts across 3 deployments with 60/25/15 ratio using weighted random selection
		// This shuffles them to reflect real-world insertion patterns
		randVal := rand.Intn(100)
		var deploymentID string
		if randVal < 60 {
			// 60% chance -> Deployment1
			deploymentID = fixtureconsts.Deployment1
		} else if randVal < 85 {
			// 25% chance (60-85) -> Deployment2
			deploymentID = fixtureconsts.Deployment2
		} else {
			// 15% chance (85-100) -> Deployment3
			deploymentID = fixtureconsts.Deployment3
		}
		a.GetDeployment().Id = deploymentID

		// Distribute alerts across real default policies
		policyInfo := defaultPolicyIDs[rand.Intn(len(defaultPolicyIDs))]
		a.Policy = fixtures.GetPolicy()
		a.Policy.Id = policyInfo.id
		a.Policy.Name = policyInfo.name
		a.Policy.Severity = policyInfo.severity

		// Set lifecycle stage and state for realistic queries
		lifecycleStages := []storage.LifecycleStage{
			storage.LifecycleStage_DEPLOY,
			storage.LifecycleStage_RUNTIME,
		}
		a.LifecycleStage = lifecycleStages[rand.Intn(len(lifecycleStages))]

		states := []storage.ViolationState{
			storage.ViolationState_ACTIVE,
			storage.ViolationState_ATTEMPTED,
		}
		a.State = states[rand.Intn(len(states))]

		sevToCount[a.GetPolicy().GetSeverity()]++
		require.NoError(b, datastore.UpsertAlert(ctx, a))
	}

	var expected []*violationsBySeverity
	for sev, count := range sevToCount {
		expected = append(expected, &violationsBySeverity{count, int(sev)})
	}

	query := pkgSearch.NewQueryBuilder().
		AddStringsHighlighted(pkgSearch.Cluster, pkgSearch.WildcardString).
		AddStringsHighlighted(pkgSearch.Category, pkgSearch.WildcardString).
		AddStringsHighlighted(pkgSearch.Severity, pkgSearch.WildcardString).
		ProtoQuery()
	b.Run("searchWithStringHighlighted", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			runSearchAndGroupResults(ctx, b, datastore, query, expected)
		}
	})

	query = pkgSearch.EmptyQuery()
	b.Run("searchWithoutHighlighted (aka get IDs only)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			runSearch(ctx, b, datastore, query)
		}
	})

	b.Run("searchWithRawListAlert", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			runSearchListAlerts(ctx, b, datastore, expected)
		}
	})

	b.Run("searchRawAlerts", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			runSearchRawAlerts(ctx, b, datastore, expected)
		}
	})

	// Real-world query patterns for SearchRawAlerts
	b.Run("searchRawAlerts/byPolicyIDAndState", func(b *testing.B) {
		// Query: specific policy ID + state (Active or Attempted)
		policyID := defaultPolicyIDs[0].id
		query := pkgSearch.NewQueryBuilder().
			AddExactMatches(pkgSearch.PolicyID, policyID).
			AddStrings(pkgSearch.ViolationState,
				storage.ViolationState_ACTIVE.String(),
				storage.ViolationState_ATTEMPTED.String()).
			ProtoQuery()
		for i := 0; i < b.N; i++ {
			results, err := datastore.SearchRawAlerts(ctx, query, false)
			require.NoError(b, err)
			require.NotNil(b, results)
		}
	})

	b.Run("searchRawAlerts/byDeploymentIDLifecycleAndState", func(b *testing.B) {
		// Query: specific deployment ID + lifecycle (runtime) + state (Active or Attempted)
		if len(deploymentIDs) > 0 {
			deploymentID := deploymentIDs[0]
			query := pkgSearch.NewQueryBuilder().
				AddExactMatches(pkgSearch.DeploymentID, deploymentID).
				AddExactMatches(pkgSearch.LifecycleStage, storage.LifecycleStage_RUNTIME.String()).
				AddStrings(pkgSearch.ViolationState,
					storage.ViolationState_ACTIVE.String(),
					storage.ViolationState_ATTEMPTED.String()).
				ProtoQuery()
			for i := 0; i < b.N; i++ {
				results, err := datastore.SearchRawAlerts(ctx, query, false)
				require.NoError(b, err)
				require.NotNil(b, results)
			}
		}
	})

	b.Run("searchRawAlerts/byDeploymentIDAndState", func(b *testing.B) {
		// Query: specific deployment ID + state (Active)
		if len(deploymentIDs) > 0 {
			deploymentID := deploymentIDs[0]
			query := pkgSearch.NewQueryBuilder().
				AddExactMatches(pkgSearch.DeploymentID, deploymentID).
				AddExactMatches(pkgSearch.ViolationState, storage.ViolationState_ACTIVE.String()).
				ProtoQuery()
			for i := 0; i < b.N; i++ {
				results, err := datastore.SearchRawAlerts(ctx, query, false)
				require.NoError(b, err)
				require.NotNil(b, results)
			}
		}
	})

	query = pkgSearch.NewQueryBuilder().
		AddSelectFields(pkgSearch.NewQuerySelect(pkgSearch.AlertID).AggrFunc(aggregatefunc.Count).Distinct()).
		AddGroupBy(pkgSearch.Severity).ProtoQuery()
	b.Run("selectQuery", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			runSelectQuery(ctx, b, testDB, query, expected)
		}
	})

	b.Run("markResolvedBatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := datastore.MarkAlertsResolvedBatch(ctx, ids...)
			require.NoError(b, err)
		}
	})

	batchSizes := []int{2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096}
	alertBatch := make([]*storage.Alert, 8192)
	for i := range alertBatch {
		alertBatch[i] = &storage.Alert{}
		require.NoError(b, testutils.FullInit(alertBatch[i], testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		alertDeployment := &storage.Alert_Deployment{}
		require.NoError(b, testutils.FullInit(alertDeployment, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		alertBatch[i].EntityType = storage.Alert_DEPLOYMENT
		alertBatch[i].Entity = &storage.Alert_Deployment_{
			Deployment: alertDeployment,
		}
	}
	for _, batchSize := range batchSizes {
		curBatch := make([]*storage.Alert, batchSize)
		for i := range batchSize {
			curBatch[i] = alertBatch[i%len(alertBatch)]
		}
		b.Run(fmt.Sprintf("UpsertMany/%d", batchSize), func(ib *testing.B) {
			for ib.Loop() {
				assert.NoError(ib, datastore.UpsertAlerts(ctx, curBatch))
			}
		})
	}
}

func runSearchAndGroupResults(ctx context.Context, t testing.TB, datastore DataStore, query *v1.Query, expected []*violationsBySeverity) {
	results, err := datastore.Search(ctx, query, true)
	require.NoError(t, err)
	require.NotNil(t, results)

	countsBySev := make([]int, len(storage.Severity_name))
	severityField := mappings.OptionsMap.MustGet(pkgSearch.Severity.String())
	for _, result := range results {
		sev := result.Matches[severityField.FieldPath][0] // Each alert has only one severity.
		countsBySev[storage.Severity_value[sev]]++
	}
	var actual []*violationsBySeverity
	for idx, count := range countsBySev {
		if count > 0 {
			actual = append(actual, &violationsBySeverity{
				AlertIDCount: count,
				Severity:     idx,
			})
		}
	}
	assert.ElementsMatch(t, expected, actual)
}

func runSearch(ctx context.Context, t testing.TB, datastore DataStore, query *v1.Query) {
	results, err := datastore.Search(ctx, query, true)
	require.NoError(t, err)
	require.NotNil(t, results)
}

func runSearchListAlerts(ctx context.Context, t testing.TB, datastore DataStore, expected []*violationsBySeverity) {
	results, err := datastore.SearchListAlerts(ctx, pkgSearch.EmptyQuery(), true)
	require.NoError(t, err)
	require.NotNil(t, results)

	countsBySev := make([]int, len(storage.Severity_name))
	for _, result := range results {
		countsBySev[result.GetPolicy().GetSeverity()]++
	}
	var actual []*violationsBySeverity
	for idx, count := range countsBySev {
		if count > 0 {
			actual = append(actual, &violationsBySeverity{
				AlertIDCount: count,
				Severity:     idx,
			})
		}
	}
	assert.ElementsMatch(t, expected, actual)
}

func runSearchRawAlerts(ctx context.Context, t testing.TB, datastore DataStore, expected []*violationsBySeverity) {
	results, err := datastore.SearchRawAlerts(ctx, pkgSearch.EmptyQuery(), true)
	require.NoError(t, err)
	require.NotNil(t, results)

	countsBySev := make([]int, len(storage.Severity_name))
	for _, result := range results {
		countsBySev[result.GetPolicy().GetSeverity()]++
	}
	var actual []*violationsBySeverity
	for idx, count := range countsBySev {
		if count > 0 {
			actual = append(actual, &violationsBySeverity{
				AlertIDCount: count,
				Severity:     idx,
			})
		}
	}
	assert.ElementsMatch(t, expected, actual)
}

func runSelectQuery(ctx context.Context, t testing.TB, testDB *pgtest.TestPostgres, q *v1.Query, expected []*violationsBySeverity) {
	results, err := postgres.RunSelectRequestForSchema[violationsBySeverity](ctx, testDB.DB, schema.AlertsSchema, q)
	require.NoError(t, err)
	assert.ElementsMatch(t, expected, results)
}

type violationsBySeverity struct {
	AlertIDCount int `db:"alert_id_count"`
	Severity     int `db:"severity"` // int because of enum
}
