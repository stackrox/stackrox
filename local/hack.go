package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/stackrox/rox/central/cve/image/datastore"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/option"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
)

func main() {
	if features.PostgresDatastore.Enabled() {

		// PREREQUISITES TO RUNNING POSTGRES
		// 1. Get a SQL dump of DB that has the data you want]
		//    kubectl exec -ti -n=stackrox $(kubectl get pod -n=stackrox -l app=central-db --output 'jsonpath={.items..metadata.name}') -- sh -c '/usr/bin/pg_dumpall' > local/central_active.sql
		// 2. Run a docker container with postgres:
		//   docker run -it -p 5432:5432 -e POSTGRES_USER=local-postgres -e POSTGRES_PASSWORD=local-pg-password postgres:14.2-alpine
		// 3. Restore the SQL dump from #1 to this local DB
		//   cat local/central_active.sql |  docker exec -i $(docker ps -lq) sh -c 'PGPASSWORD=local-pg-password psql -h localhost -U local-postgres -p 5432'
		// 4. Go to pkg/config/config.go, and update the variable "defaultDBSource" to:
		//		host=localhost port=5432 user=local-postgres sslmode=disable statement_timeout=600000 pool_min_conns=1 pool_max_conns=90 client_encoding=UTF-8
		// TODO: Expose a nicer way to set the DB connection without having to manually modify it

		// Set a temp password file that the postgres package can read from
		f, err := os.CreateTemp("", "local-pg-pwd")
		if err != nil {
			panic(err)
		}
		defer func() {
			_ = os.Remove(f.Name())
		}()
		_, err = f.WriteString("local-pg-password")
		if err != nil {
			panic(err)
		}
		pgconfig.DBPasswordFile = f.Name()

		ctx := context.Background()
		pool := globaldb.InitializePostgres(ctx)
		if pool == nil {
			panic("Failed to initialize postgres")
		}

		// Uncomment this if you want every query run to be logged out to stdout of your postgres container
		// pool.Exec(ctx, "SET log_statement = 'all'")
	} else {
		option.CentralOptions.DBPathBase = "local/database-restore/full"

		blevePath := filepath.Join(option.CentralOptions.DBPathBase, "bleve")
		globalindex.DefaultBlevePath = filepath.Join(blevePath, "default")
		globalindex.DefaultTmpBlevePath = filepath.Join(blevePath, "tmp")
		globalindex.SeparateIndexPath = filepath.Join(blevePath, "separate")
	}

	// Can start accessing _most_ singletons. Some singletons that access certificates will fail
	ds := datastore.Singleton()
	count, err := ds.Count(sac.WithAllAccess(context.Background()), search.NewQueryBuilder().ProtoQuery())
	if err != nil {
		panic(err)
	}
	fmt.Printf("Found %v CVEs in store", count)
}
