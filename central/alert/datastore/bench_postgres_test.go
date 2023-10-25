//go:build sql_integration

package datastore

import (
	"context"
	"math/rand"
	"testing"

	"github.com/stackrox/rox/central/alert/mappings"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkAlertDatabaseOps(b *testing.B) {
	testDB := pgtest.ForT(b)
	ctx := sac.WithAllAccess(context.Background())
	datastore, err := GetTestPostgresDataStore(b, testDB.DB)
	require.NoError(b, err)

	var ids []string
	sevToCount := make(map[storage.Severity]int)
	// Keep the count low in CI. You can run w/ higher numbers locally.
	for i := 0; i < 1000; i++ {
		id := uuid.NewV4().String()
		ids = append(ids, id)
		a := fixtures.GetAlertWithID(id)
		a.Policy.Severity = storage.Severity(rand.Intn(5))
		sevToCount[a.Policy.Severity]++
		require.NoError(b, datastore.UpsertAlert(ctx, a))
	}
	log.Info("Successfully loaded the DB")

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

	query = pkgSearch.NewQueryBuilder().
		AddSelectFields(pkgSearch.NewQuerySelect(pkgSearch.AlertID).AggrFunc(aggregatefunc.Count).Distinct()).
		AddGroupBy(pkgSearch.Severity).ProtoQuery()
	b.Run("selectQuery", func(b *testing.B) {
		b.Skip("ROX-20480: This test is failing. Skipping!")
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
}

func runSearchAndGroupResults(ctx context.Context, t testing.TB, datastore DataStore, query *v1.Query, expected []*violationsBySeverity) {
	results, err := datastore.Search(ctx, query)
	require.NoError(t, err)
	require.NotNil(t, results)

	countsBySev := make([]int, len(expected))
	severityField := mappings.OptionsMap.MustGet(pkgSearch.Severity.String())
	for _, result := range results {
		sev := result.Matches[severityField.FieldPath][0] // Each alert has only one severity.
		countsBySev[storage.Severity_value[sev]]++
	}
	var actual []*violationsBySeverity
	for idx, count := range countsBySev {
		actual = append(actual, &violationsBySeverity{
			AlertIDCount: count,
			Severity:     idx,
		})
	}
	assert.ElementsMatch(t, expected, actual)
}

func runSearch(ctx context.Context, t testing.TB, datastore DataStore, query *v1.Query) {
	results, err := datastore.Search(ctx, query)
	require.NoError(t, err)
	require.NotNil(t, results)
}

func runSearchListAlerts(ctx context.Context, t testing.TB, datastore DataStore, expected []*violationsBySeverity) {
	results, err := datastore.SearchListAlerts(ctx, pkgSearch.EmptyQuery())
	require.NoError(t, err)
	require.NotNil(t, results)

	countsBySev := make([]int, len(expected))
	for _, result := range results {
		countsBySev[result.GetPolicy().GetSeverity()]++
	}
	var actual []*violationsBySeverity
	for idx, count := range countsBySev {
		actual = append(actual, &violationsBySeverity{
			AlertIDCount: count,
			Severity:     idx,
		})
	}
	assert.ElementsMatch(t, expected, actual)
}

func runSelectQuery(ctx context.Context, t testing.TB, testDB *pgtest.TestPostgres, q *v1.Query, expected []*violationsBySeverity) {
	results, err := postgres.RunSelectRequestForSchema[violationsBySeverity](ctx, testDB.DB, schema.AlertsSchema, q)
	require.NoError(t, err)
	assert.ElementsMatch(t, expected, results)
}

type violationsBySeverity struct {
	AlertIDCount int `db:"alertidcount"`
	Severity     int `db:"severity"` // int because of enum
}
