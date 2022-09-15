package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/cve/image/datastore"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/option"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest/conn"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"k8s.io/utils/env"
)

func main() {
	if features.PostgresDatastore.Enabled() {

		// PREREQUISITES TO RUNNING POSTGRES
		// 1. Get a SQL dump of DB that has the data you want]
		//    kubectl exec -ti -n=stackrox $(kubectl get pod -n=stackrox -l app=central-db --output 'jsonpath={.items..metadata.name}') -- sh -c '/usr/bin/pg_dumpall' > local/central_active.sql
		// 2. Run a docker container with postgres:
		//   docker run -it -p 5432:5432 -e POSTGRES_USER=${USER} -e POSTGRES_HOST_AUTH_METHOD=trust postgres:14.2-alpine
		// 3. Restore the SQL dump from #1 to this local DB
		//   cat local/central_active.sql |  docker exec -i $(docker ps -lq) sh -c 'psql -h localhost -U '$USER' -p 5432'
		// 4. Update central/globaldb/postgres.go and add the following function:
		//		func SetPostgres(pool *pgxpool.Pool) {
		//			postgresDB = pool
		//		}
		// TODO: Expose a nicer way to set the DB connection without having to manually add the function

		source := conn.GetConnectionStringWithDatabaseName(env.GetString("POSTGRES_DB", "central_active"))
		config, err := pgxpool.ParseConfig(source)
		if err != nil {
			panic(err)
		}

		ctx := context.Background()
		pool, err := pgxpool.ConnectConfig(ctx, config)
		if err != nil {
			panic(err)
		}
		globaldb.SetPostgres(pool)

		// Uncomment this if you want every query run to be logged out to stdout of your postgres container
		//pool.Exec(ctx, "SET log_statement = 'all'")
	} else {
		option.CentralOptions.DBPathBase = "local/database-restore/full"

		blevePath := filepath.Join(option.CentralOptions.DBPathBase, "bleve")
		globalindex.DefaultBlevePath = filepath.Join(blevePath, "default")
		globalindex.DefaultTmpBlevePath = filepath.Join(blevePath, "tmp")
		globalindex.SeparateIndexPath = filepath.Join(blevePath, "separate")
	}

	//Can start accessing _most_ singletons. Some singletons that access certificates will fail
	ds := datastore.Singleton()
	count, err := ds.Count(sac.WithAllAccess(context.Background()), search.NewQueryBuilder().ProtoQuery())
	if err != nil {
		panic(err)
	}
	fmt.Printf("Found %v CVEs in store", count)
}
