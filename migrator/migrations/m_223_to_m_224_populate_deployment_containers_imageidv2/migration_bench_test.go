//go:build sql_integration

package m223tom224

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_223_to_m_224_populate_deployment_containers_imageidv2/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	benchNumDeployments   = 100
	benchContainersPerDep = 5
	benchTotalContainers  = benchNumDeployments * benchContainersPerDep
)

type benchData struct {
	depIDs     []string
	serialized [][]byte // original serialized blobs without id_v2
}

func setupBench(b *testing.B, ctx context.Context, db *pghelper.TestPostgres) *benchData {
	pgutils.CreateTableFromModel(ctx, db.GetGormDB(), schema.CreateTableDeploymentsStmt)

	data := &benchData{}
	for i := 0; i < benchNumDeployments; i++ {
		dep := &storage.Deployment{
			Id:        uuid.NewV4().String(),
			Name:      fmt.Sprintf("dep-%d", i),
			Type:      "Deployment",
			Namespace: "default",
		}
		for j := 0; j < benchContainersPerDep; j++ {
			hash := fmt.Sprintf("%064x", i*benchContainersPerDep+j)
			dep.Containers = append(dep.GetContainers(), &storage.Container{
				Image: &storage.ContainerImage{
					Id:   "sha256:" + hash,
					Name: &storage.ImageName{FullName: fmt.Sprintf("registry.example.com/image-%d:%d@sha256:%s", i, j, hash)},
				},
			})
		}

		serialized, err := dep.MarshalVT()
		if err != nil {
			b.Fatal(err)
		}
		data.depIDs = append(data.depIDs, dep.GetId())
		data.serialized = append(data.serialized, serialized)

		_, err = db.Exec(ctx,
			"INSERT INTO deployments (id, name, type, namespace, serialized) VALUES ($1, $2, $3, $4, $5)",
			pgutils.NilOrUUID(dep.GetId()), dep.GetName(), dep.GetType(), dep.GetNamespace(), serialized)
		if err != nil {
			b.Fatal(err)
		}

		for idx, c := range dep.GetContainers() {
			_, err = db.Exec(ctx,
				"INSERT INTO deployments_containers (deployments_id, idx, image_id, image_name_fullname) VALUES ($1, $2, $3, $4)",
				pgutils.NilOrUUID(dep.GetId()), idx, c.GetImage().GetId(), c.GetImage().GetName().GetFullName())
			if err != nil {
				b.Fatal(err)
			}
		}
	}
	return data
}

func resetBench(b *testing.B, ctx context.Context, db *pghelper.TestPostgres, data *benchData) {
	// Reset image_idv2 columns.
	if _, err := db.Exec(ctx, "UPDATE deployments_containers SET image_idv2 = ''"); err != nil {
		b.Fatal(err)
	}
	// Reset serialized blobs to original (without id_v2).
	for i, id := range data.depIDs {
		if _, err := db.Exec(ctx, "UPDATE deployments SET serialized = $1 WHERE id = $2",
			data.serialized[i], pgutils.NilOrUUID(id)); err != nil {
			b.Fatal(err)
		}
	}
}

func verifyBench(b *testing.B, ctx context.Context, db *pghelper.TestPostgres) {
	var count int
	err := db.QueryRow(ctx,
		"SELECT COUNT(*) FROM deployments_containers WHERE image_idv2 != '' AND image_idv2 IS NOT NULL").Scan(&count)
	if err != nil {
		b.Fatal(err)
	}
	if count != benchTotalContainers {
		b.Fatalf("expected %d containers with image_idv2, got %d", benchTotalContainers, count)
	}
}

func benchMigrate(b *testing.B, migrateFn func(*types.Databases) error) {
	ctx := sac.WithAllAccess(context.Background())
	db := pghelper.ForT(b, false)
	data := setupBench(b, ctx, db)

	batchSize = 5000
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		resetBench(b, ctx, db, data)
		b.StartTimer()

		dbs := &types.Databases{
			GormDB:     db.GetGormDB(),
			PostgresDB: db.DB,
			DBCtx:      ctx,
		}
		if err := migrateFn(dbs); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()

	verifyBench(b, ctx, db)
}

func BenchmarkMigration(b *testing.B) {
	benchMigrate(b, migrate)
}
