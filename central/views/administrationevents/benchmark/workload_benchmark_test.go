//go:build sql_integration

package benchmark_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/central/views/administrationevents"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type workloadBenchmarkSize struct {
	events, deployments, images, matchingImages int
}

type workloadBenchmark struct {
	db             postgres.DB
	base, workload *v1.Query
	expectedCount  int
}

// BenchmarkWorkloadFilter compares event-view reads, including query construction,
// SQL compilation, database round trips and event decoding. The two-query strategy selects DISTINCT
// image IDs before building the event query; it does not deserialize deployments.
// The search compiler uses IN or an array parameter with ANY for the second query,
// depending on the number of IDs.
// B/op and allocs/op measure Go allocations, not PostgreSQL memory or peak RSS.
// Run with -run '^$' -bench '^BenchmarkWorkloadFilter$' -benchmem -count=3.
func BenchmarkWorkloadFilter(b *testing.B) {
	for name, size := range map[string]workloadBenchmarkSize{
		"1000_events_10_images":       {1000, 1000, 100, 10},
		"10000_events_250_images":     {10000, 10000, 2500, 250},
		"100000_events_1000_images":   {100000, 20000, 10000, 1000},
		"100000_events_10000_images":  {100000, 20000, 10000, 10000},
		"100000_events_100000_images": {100000, 200000, 100000, 100000},
	} {
		b.Run(name, func(b *testing.B) {
			fixture := newWorkloadBenchmark(b, size)
			ctx := sac.WithAllAccess(context.Background())
			for _, read := range []string{"count", "page"} {
				b.Run(read, func(b *testing.B) {
					for _, materialize := range []bool{false, true} {
						strategy := "exists"
						if materialize {
							strategy = "image_ids_in_go"
						}
						b.Run(strategy, func(b *testing.B) {
							expected := fixture.expectedCount
							if read == "page" {
								expected = min(expected, 50)
							}
							actual, err := fixture.run(ctx, read, materialize)
							require.NoError(b, err)
							require.Equal(b, expected, actual)
							b.ReportAllocs()
							b.ResetTimer()
							for b.Loop() {
								actual, err := fixture.run(ctx, read, materialize)
								if err != nil {
									b.Fatal(err)
								}
								if actual != expected {
									b.Fatalf("got %d events, want %d", actual, expected)
								}
							}
						})
					}
				})
			}
		})
	}
}

// TestWorkloadFilterQueryPlan explains the actual SQL emitted by the event view.
// Run with -v -count=1 to see the plan, actual row counts, loops and buffer usage.
// Plans are observations, not assertions: PostgreSQL may choose different plans
// as its version, table statistics, selectivity or available memory change.
func TestWorkloadFilterQueryPlan(t *testing.T) {
	for name, size := range map[string]workloadBenchmarkSize{
		"small":           {1000, 1000, 100, 10},
		"medium":          {10000, 10000, 2500, 250},
		"large selective": {100000, 20000, 10000, 1000},
		"large broad":     {100000, 20000, 10000, 10000},
		"100k images":     {100000, 200000, 100000, 100000},
	} {
		t.Run(name, func(t *testing.T) {
			testWorkloadFilterQueryPlan(t, size)
		})
	}
}

func testWorkloadFilterQueryPlan(t *testing.T, size workloadBenchmarkSize) {
	t.Helper()
	fixture := newWorkloadBenchmark(t, size)
	ctx := sac.WithAllAccess(context.Background())
	t.Logf("%d events (10%% non-image), %d deployments, two containers per deployment, %d matching distinct images",
		size.events, size.deployments, size.matchingImages)
	for _, read := range []string{"count", "page"} {
		t.Run(read, func(t *testing.T) {
			recorder := &workloadQueryRecorder{DB: fixture.db}
			recorded := *fixture
			recorded.db = recorder
			expected := fixture.expectedCount
			if read == "page" {
				expected = min(expected, 50)
			}
			// PostgreSQL can switch from custom to generic plans after five uses.
			// Warm the driver's prepared statement before explaining that statement.
			for range 6 {
				actual, err := recorded.run(ctx, read, false)
				require.NoError(t, err)
				require.Equal(t, expected, actual)
			}
			actual, err := fixture.run(ctx, read, true)
			require.NoError(t, err)
			require.Equal(t, expected, actual)
			logWorkloadPlan(t, ctx, fixture.db, recorder.sql, recorder.args)
		})
	}
	query, err := fixture.query(ctx, false)
	require.NoError(t, err)
	query.Pagination = search.NewPagination().Limit(50).
		AddSortOption(search.NewSortOption(search.LastUpdatedTime).Reversed(true)).Proto()
	pagination := query.GetPagination().CloneVT()
	expected, err := administrationevents.NewEventView(fixture.db).Get(ctx, query)
	require.NoError(t, err)
	query, err = fixture.query(ctx, true)
	require.NoError(t, err)
	query.Pagination = pagination
	actual, err := administrationevents.NewEventView(fixture.db).Get(ctx, query)
	require.NoError(t, err)
	require.Equal(t, protoutils.GetIDs(expected), protoutils.GetIDs(actual))
}

// logWorkloadPlan explains the statement already prepared on the fixture's single connection.
func logWorkloadPlan(t *testing.T, ctx context.Context, db postgres.DB, sql string, args []any) {
	t.Helper()
	t.Logf("SQL: %s\nParameters: %#v", sql, args)
	var statement string
	var genericPlans, customPlans int64
	require.NoError(t, db.QueryRow(ctx,
		"SELECT name, generic_plans, custom_plans FROM pg_prepared_statements WHERE statement = $1", sql).
		Scan(&statement, &genericPlans, &customPlans))
	t.Logf("prepared statement: generic_plans=%d custom_plans=%d", genericPlans, customPlans)
	// Ask PostgreSQL to quote fixture values for EXPLAIN EXECUTE.
	arguments := make([]string, 0, len(args))
	for _, arg := range args {
		var quoted string
		require.NoError(t, db.QueryRow(ctx, "SELECT quote_literal($1::text)", arg).Scan(&quoted))
		arguments = append(arguments, quoted)
	}
	explain := "EXPLAIN (ANALYZE, BUFFERS, SETTINGS) EXECUTE " + pgx.Identifier{statement}.Sanitize() +
		"(" + strings.Join(arguments, ", ") + ")"
	rows, err := db.Query(ctx, explain)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var line string
		require.NoError(t, rows.Scan(&line))
		t.Log(line)
	}
	require.NoError(t, rows.Err())
}

func (f *workloadBenchmark) run(ctx context.Context, read string, materialize bool) (int, error) {
	query, err := f.query(ctx, materialize)
	if err != nil {
		return 0, err
	}
	view := administrationevents.NewEventView(f.db)
	if read == "count" {
		return view.Count(ctx, query)
	}
	query.Pagination = search.NewPagination().Limit(50).
		AddSortOption(search.NewSortOption(search.LastUpdatedTime).Reversed(true)).Proto()
	events, err := view.Get(ctx, query)
	return len(events), err
}

func (f *workloadBenchmark) query(ctx context.Context, materialize bool) (*v1.Query, error) {
	if !materialize {
		return administrationevents.WithWorkloadQuery(f.base, f.workload)
	}
	workload := f.workload.CloneVT()
	workload.Selects = []*v1.QuerySelect{search.NewQuerySelect(search.ImageID).Distinct().Proto()}
	var imageIDs []string
	err := pgSearch.RunSelectRequestForSchemaFn(ctx, f.db, schema.DeploymentsSchema, workload, func(row *struct {
		ImageID string `db:"image_id"`
	}) error {
		if row.ImageID != "" {
			imageIDs = append(imageIDs, row.ImageID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	matches := search.MatchNoneQuery()
	if len(imageIDs) != 0 {
		matches = search.NewQueryBuilder().AddExactMatches(search.ResourceID, imageIDs...).ProtoQuery()
	}
	imageType := search.NewQueryBuilder().AddExactMatches(search.ResourceType, "Image").ProtoQuery()
	otherTypes := search.NewBooleanQuery(nil, &v1.DisjunctionQuery{Queries: []*v1.Query{imageType}})
	return search.ConjunctionQuery(f.base.CloneVT(),
		search.DisjunctionQuery(search.ConjunctionQuery(imageType, matches), otherTypes)), nil
}

func newWorkloadBenchmark(tb testing.TB, size workloadBenchmarkSize) *workloadBenchmark {
	tb.Helper()
	db := pgtest.ForT(tb)
	ctx := sac.WithAllAccess(context.Background())
	var version, workMem, jit, sharedBuffers, cacheMode string
	require.NoError(tb, db.QueryRow(ctx,
		"SELECT version(), current_setting('work_mem'), current_setting('jit'), current_setting('shared_buffers'), current_setting('plan_cache_mode')").
		Scan(&version, &workMem, &jit, &sharedBuffers, &cacheMode))
	tb.Logf("%s; work_mem=%s; jit=%s; shared_buffers=%s; plan_cache_mode=%s; flattened images=%v",
		version, workMem, jit, sharedBuffers, cacheMode, features.FlattenImageData.Enabled())
	connection, err := db.Acquire(ctx)
	require.NoError(tb, err)
	defer connection.Release()
	deploymentIDs := make([]string, size.deployments)
	for i := range deploymentIDs {
		deploymentIDs[i] = uuid.NewV4().String()
	}
	_, err = connection.CopyFrom(ctx, pgx.Identifier{"deployments"},
		[]string{"id", "name", "namespace"}, pgx.CopyFromSlice(size.deployments, func(i int) ([]any, error) {
			namespace := "other"
			if i%size.images < size.matchingImages {
				namespace = "payments"
			}
			return []any{deploymentIDs[i], "app", namespace}, nil
		}))
	require.NoError(tb, err)
	_, err = connection.CopyFrom(ctx, pgx.Identifier{"deployments_containers"},
		[]string{"deployments_id", "idx", "image_id", "image_idv2"},
		pgx.CopyFromSlice(size.deployments*2, func(i int) ([]any, error) {
			imageID := fmt.Sprintf("image-%08d", (i/2)%size.images)
			return []any{deploymentIDs[i/2], i % 2, imageID, imageID}, nil
		}))
	require.NoError(tb, err)
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	message := strings.Repeat("image scan failed ", 64)
	var expectedCount int
	_, err = connection.CopyFrom(ctx, pgx.Identifier{"administration_events"},
		[]string{"id", "type", "level", "domain", "resource_type", "resource_id", "createdat", "lastoccurredat", "serialized"},
		pgx.CopyFromSlice(size.events, func(i int) ([]any, error) {
			resourceType := "Image"
			resourceID := fmt.Sprintf("image-%08d", i%size.images)
			if i%10 == 0 {
				resourceType = "Node"
				resourceID = fmt.Sprintf("node-%08d", i)
			}
			if resourceType != "Image" || i%size.images < size.matchingImages {
				expectedCount++
			}
			occurred := start.Add(time.Duration(i) * time.Second)
			event := &storage.AdministrationEvent{
				Id: uuid.NewV4().String(), Message: message, Domain: "Image Scanning",
				Type:      storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE,
				Level:     storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
				Resource:  &storage.AdministrationEvent_Resource{Type: resourceType, Id: resourceID},
				CreatedAt: timestamppb.New(occurred), LastOccurredAt: timestamppb.New(occurred),
			}
			data, err := event.MarshalVT()
			if err != nil {
				return nil, err
			}
			return []any{event.GetId(), int32(event.GetType()), int32(event.GetLevel()), event.GetDomain(),
				resourceType, resourceID, occurred, occurred, data}, nil
		}))
	require.NoError(tb, err)
	for _, table := range []string{"administration_events", "deployments", "deployments_containers"} {
		_, err = connection.Exec(ctx, "ANALYZE "+table)
		require.NoError(tb, err)
	}
	return &workloadBenchmark{
		db:   db,
		base: search.NewQueryBuilder().AddTimeRangeField(search.CreatedTime, start, start.AddDate(1, 0, 0)).ProtoQuery(),
		workload: search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "app").
			AddExactMatches(search.Namespace, "payments").ProtoQuery(),
		expectedCount: expectedCount,
	}
}

type workloadQueryRecorder struct {
	postgres.DB
	sql  string
	args []any
}

func (r *workloadQueryRecorder) Query(ctx context.Context, sql string, args ...any) (*postgres.Rows, error) {
	r.sql, r.args = sql, args
	return r.DB.Query(ctx, sql, args...)
}

func (r *workloadQueryRecorder) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	r.sql, r.args = sql, args
	return r.DB.QueryRow(ctx, sql, args...)
}
