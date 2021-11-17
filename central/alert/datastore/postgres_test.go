package datastore

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/lib/pq"
	alertPGIndex "github.com/stackrox/rox/central/alert/datastore/internal/index/postgres"
	searcher "github.com/stackrox/rox/central/alert/datastore/internal/search"
	alertPGStore "github.com/stackrox/rox/central/alert/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/alert/mappings"
	"github.com/stackrox/rox/central/globaldb"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres"
)

/*
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

*/

func TestPGX(t *testing.T) {
	conn, err := pgx.Connect(context.Background(), "host=localhost port=5432 user=postgres sslmode=disable statement_timeout=60000")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	if err := conn.Ping(context.Background()); err != nil {
		panic(err)
	}
}

func convertEnumSliceToIntArray(i interface{}) []int32 {
	enumSlice := reflect.ValueOf(i)
	enumSliceLen := enumSlice.Len()
	resultSlice := make([]int32, 0, enumSliceLen)
	for i := 0; i < enumSlice.Len(); i++ {
		resultSlice = append(resultSlice, int32(enumSlice.Index(i).Int()))
	}
	return resultSlice
}

func TestSliceConversion(t *testing.T) {
	stages := []storage.LifecycleStage {storage.LifecycleStage_BUILD, storage.LifecycleStage_DEPLOY}

	//l := int(storage.LifecycleStage_BUILD)

	lol := convertEnumSliceToIntArray(stages)
	fmt.Println(lol)
}

func TestT(t *testing.T) {
	config, err := pgxpool.ParseConfig("pool_min_conns=100 pool_max_conns=100 host=localhost port=5432 database=postgres user=connorgorman sslmode=disable statement_timeout=60000")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", config)

	db, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	alertStore := alertPGStore.NewFullStore(db)
	fmt.Println(alertStore)

	alertIndex := alertPGIndex.NewIndexer(db)
	fmt.Println(alertIndex)

	searcher := searcher.New(alertStore, alertIndex)

	alert := fixtures.GetAlertWithID("0098077c-d872-4bf9-a8b2-666f9f8e939a")
	if err := alertStore.Upsert(alert); err != nil {
		panic(err)
	}

	qb := search.NewQueryBuilder().
		AddStrings(
			search.ViolationState,
			storage.ViolationState_ACTIVE.String(),
			storage.ViolationState_ATTEMPTED.String()).
		AddStrings(search.Cluster, "prod").
		AddStringsHighlighted(search.ClusterID, search.WildcardString)

	pq := qb.ProtoQuery()

	pq.Pagination = &v1.QueryPagination{
		Offset: 0,
		Limit: 50,
	}

	results, err := searcher.SearchRawAlerts(context.Background(), pq)
	if err != nil {
		panic(err)
	}
	for _, r := range results {
		fmt.Printf("result: %+v %+v\n", r.GetDeployment().GetId(), r.GetPolicy().GetId())
	}
}

func TestDelete(t *testing.T) {
	config, err := pgxpool.ParseConfig("pool_min_conns=100 pool_max_conns=100 host=localhost port=5432 user=postgres sslmode=disable statement_timeout=60000")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", config)

	db, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	alertStore := alertPGStore.NewFullStore(db)
	fmt.Println(alertStore)

	alertIndex := alertPGIndex.NewIndexer(db)
	fmt.Println(alertIndex)

	searcher := searcher.New(alertStore, alertIndex)

	qb := search.NewQueryBuilder().
		AddExactMatches(search.DeploymentID, "8861671b-90b9-4edc-a42e-574b2dc65c18").
		AddExactMatches(search.PolicyID, "dce17697-1b72-49d2-b18a-05d893cd9368")

	pq := qb.ProtoQuery()

	pq.Pagination = &v1.QueryPagination{
		Limit: 10,
	}

	results, err := searcher.SearchRawAlerts(context.Background(), pq)
	if err != nil {
		panic(err)
	}
	for _, r := range results {
		fmt.Printf("result: %+v %+v\n", r.GetDeployment().GetId(), r.GetPolicy().GetId())
	}
	err = postgres.RunSearchRequestDelete(v1.SearchCategory_ALERTS, pq, globaldb.GetPostgresDB(), mappings.OptionsMap)
	if err != nil {
		panic(err)
	}

	results, err = searcher.SearchRawAlerts(context.Background(), pq)
	if err != nil {
		panic(err)
	}
	for _, r := range results {
		fmt.Printf("after delete result: %+v %+v\n", r.GetDeployment().GetId(), r.GetPolicy().GetId())
	}
}

func BenchmarkGets(b *testing.B) {
	config, err := pgxpool.ParseConfig("pool_min_conns=100 pool_max_conns=100 host=localhost port=5432 user=postgres sslmode=disable statement_timeout=60000")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", config)

	db, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	alertStore := alertPGStore.NewFullStore(db)
	fmt.Println(alertStore)

	alertIndex := alertPGIndex.NewIndexer(db)
	fmt.Println(alertIndex)

	searcher := searcher.New(alertStore, alertIndex)

	qb := search.NewQueryBuilder().
		AddStrings(
			search.ViolationState,
			storage.ViolationState_ACTIVE.String(),
			storage.ViolationState_ATTEMPTED.String()).
		AddStrings(search.Cluster, "remote").
		AddStringsHighlighted(search.ClusterID, search.WildcardString)

	pq := qb.ProtoQuery()

	pq.Pagination = &v1.QueryPagination{
		Limit: 10,
	}

	ctx := sac.WithAllAccess(context.Background())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = searcher.SearchRawAlerts(ctx, pq)
		if err != nil {
			panic(err)
		}
	}
	//
	//for _, r := range results {
	//	fmt.Printf("result: %+v\n", r.GetId())
	//}
}
