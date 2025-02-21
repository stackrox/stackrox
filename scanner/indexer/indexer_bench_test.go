//go:build scanner_indexer_integration

package indexer

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/scanner/config"
	"github.com/stackrox/rox/scanner/datastore/postgres"
	"github.com/stretchr/testify/require"
)

// //go:embed testdata/images.json
var contents []byte

func indexImages(b *testing.B, indexer Indexer, images []string, n int) {
	imagesC := make(chan string)
	go func() {
		for _, image := range images[:n] {
			imagesC <- image
		}
		close(imagesC)
	}()

	var wg sync.WaitGroup
	var errs atomic.Int64
	for range 30 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for image := range imagesC {
				id := fmt.Sprintf("/v4/containerimage/%s", image)
				_, err := indexer.IndexContainerImage(context.Background(), id, "https://"+image)
				if err != nil {
					errs.Add(1)
				}
			}
		}()
	}
	wg.Wait()

	fmt.Printf("Number of errors: %d\n", errs.Load())
}

func BenchmarkIndexContainerImage(b *testing.B) {
	dbname := "index_container_image_benchmark"
	_, cleanup := testDB(b, context.Background(), dbname)
	defer cleanup()

	indexer, err := NewIndexer(context.Background(), config.IndexerConfig{
		Database: config.Database{
			ConnString: fmt.Sprintf("host=127.0.0.1 port=5432 dbname=%s sslmode=disable", dbname),
		},
		Enable:             true,
		GetLayerTimeout:    config.Duration(1 * time.Minute),
		RepositoryToCPEURL: "https://security.access.redhat.com/data/metrics/repository-to-cpe.json",
		NameToReposURL:     "https://security.access.redhat.com/data/metrics/container-name-repos-map.json",
	})
	require.NoError(b, err)

	var images []string
	err = json.Unmarshal(contents, &images)
	require.NoError(b, err)

	b.ResetTimer()
	for range b.N {
		indexImages(b, indexer, images, 100)
	}
}

func BenchmarkGCManifests(b *testing.B) {
	dbname := "gc_manifests_benchmark"
	_, cleanup := testDB(b, context.Background(), dbname)
	defer cleanup()

	indexer, err := NewIndexer(context.Background(), config.IndexerConfig{
		Database: config.Database{
			ConnString: fmt.Sprintf("host=127.0.0.1 port=5432 dbname=%s sslmode=disable", dbname),
		},
		Enable:             true,
		GetLayerTimeout:    config.Duration(1 * time.Minute),
		RepositoryToCPEURL: "https://security.access.redhat.com/data/metrics/repository-to-cpe.json",
		NameToReposURL:     "https://security.access.redhat.com/data/metrics/container-name-repos-map.json",
	})
	require.NoError(b, err)

	var images []string
	err = json.Unmarshal(contents, &images)
	require.NoError(b, err)

	indexImages(b, indexer, images, 100)

	b.ResetTimer()
	for range b.N {
		//require.NoError(b, indexer.(*localIndexer).manifestManager.RunFullGC(context.Background()))
	}
}

func testDB(t *testing.B, ctx context.Context, name string) (*pgxpool.Pool, func()) {
	t.Helper()

	pgConn := "postgresql://postgres@127.0.0.1:5432/postgres?sslmode=disable"
	pgPool, err := postgres.Connect(ctx, pgConn, name)
	require.NoError(t, err)
	createDatabase := `CREATE DATABASE ` + name
	_, err = pgPool.Exec(ctx, createDatabase)
	require.NoError(t, err)

	dbConn := fmt.Sprintf("postgresql://postgres@127.0.0.1:5432/%s?sslmode=disable", name)
	dbPool, err := postgres.Connect(ctx, dbConn, name)
	require.NoError(t, err)

	return dbPool, func() {
		dbPool.Close()
		dropDatabase := `DROP DATABASE IF EXISTS ` + name
		_, _ = pgPool.Exec(ctx, dropDatabase)
		pgPool.Close()
	}
}
