package datastore

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/lib/pq"
	alertPGIndex "github.com/stackrox/rox/central/alert/datastore/internal/index/postgres"
	alertPGStore "github.com/stackrox/rox/central/alert/datastore/internal/store/postgres"
	"github.com/stackrox/rox/pkg/fixtures"
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

func TestT(t *testing.T) {
	config, err := pgxpool.ParseConfig("pool_min_conns=100 pool_max_conns=100 host=localhost port=5432 user=postgres sslmode=disable statement_timeout=60000")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", config)

	db, err := pgxpool.ConnectConfig(context.Background(),  config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	alertStore := alertPGStore.New(db)
	fmt.Println(alertStore)

	alertIndex := alertPGIndex.NewIndexer(db)
	fmt.Println(alertIndex)

	alert := fixtures.GetAlert()

	for i := 0; i < 10; i++ {
		now := time.Now()
		if err := alertStore.Upsert(alert); err != nil {
			panic(err)
		}
		fmt.Printf("Milliseconds: %v\n", time.Since(now).Milliseconds())
	}

	fetched, _, err := alertStore.Get(alert.GetId())
	if err != nil {
		panic(err)
	}

	if !proto.Equal(fetched, alert){
		panic("nooo")
	}

	alert.Policy.Name = "heyeeyeyeyeye"
	if err := alertStore.Upsert(alert); err != nil {
		panic(err)
	}
	fetched, _, err = alertStore.Get(alert.GetId())
	if err != nil {
		panic(err)
	}

	if !proto.Equal(fetched, alert){
		panic("nooo")
	}


	stat := db.Stat()
	fmt.Println("AcquireCount", stat.AcquireCount())
	fmt.Println("AcquireDuration", stat.AcquireDuration())
	fmt.Println("AcquiredConns", stat.AcquiredConns())
	fmt.Println("CanceledAcquireCount", stat.CanceledAcquireCount())
	fmt.Println("ConstructingConns", stat.ConstructingConns())
	fmt.Println("EmptyAcquireCount", stat.EmptyAcquireCount())
	fmt.Println("IdleConns", stat.IdleConns())
	fmt.Println("MaxConns", stat.MaxConns())
	fmt.Println("TotalConns", stat.TotalConns())

	//
	//alerts, missing, err := alertStore.GetMany([]string{"0e5970b5-0fa3-4e46-9e17-1fc28b855cd8", "d71afb50-51a5-40ae-932e-68b80cd96687"})
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println(alerts, missing)
	//
	//qb := search.NewQueryBuilder().
	//	AddStrings(
	//		search.ViolationState,
	//		storage.ViolationState_ACTIVE.String(),
	//		storage.ViolationState_ATTEMPTED.String()).
	//	AddStrings(search.Cluster, "remote")
	//
	//pq := qb.ProtoQuery()
	//pq.Pagination = &v1.QueryPagination{
	//	SortOptions: []*v1.QuerySortOption{
	//		{
	//			Field:    search.ViolationTime.String(),
	//			Reversed: true,
	//		},
	//		{
	//			Field:    search.LifecycleStage.String(),
	//			Reversed: true,
	//		},
	//	},
	//}
	//
	//results, err := alertIndex.Search(pq, nil)
	//if err != nil {
	//	panic(err)
	//}
	//for _, r := range results {
	//	fmt.Println("result:", r.ID)
	//}
}
