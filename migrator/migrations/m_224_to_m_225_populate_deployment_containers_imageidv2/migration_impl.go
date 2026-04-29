package m224tom225

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_224_to_m_225_populate_deployment_containers_imageidv2/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	log       = logging.LoggerForModule()
	batchSize = 5000
)

type batchResult struct {
	updated int
	lastID  string
}

// migrate updates image_idv2 in both the deployments_containers column and the
// deployments.serialized proto blob. Updating the blob is necessary because
// the platform reprocessor reads the blob and upserts back — if the blob
// doesn't have id_v2, the reprocessor clobbers the column value.
func migrate(database *types.Databases) error {
	pgutils.CreateTableFromModel(database.DBCtx, database.GormDB, schema.CreateTableDeploymentsStmt)
	ctx := database.DBCtx

	conn, err := database.PostgresDB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	updatedCount := 0

	// Paginate by deployment ID to guarantee forward progress even if some
	// deployments consistently fail to update.
	lastID := uuid.Nil.String()
	for {
		res, err := migrateBatch(ctx, conn, lastID)
		if err != nil {
			return err
		}
		if res.lastID == "" {
			break
		}
		lastID = res.lastID
		updatedCount += res.updated
	}

	log.Infof("Populated image_idv2 for %d deployments", updatedCount)
	return nil
}

type depRow struct {
	id         string
	serialized []byte
}

func migrateBatch(ctx context.Context, conn *postgres.Conn, lastID string) (batchResult, error) {
	deps, err := fetchDeployments(ctx, conn, lastID)
	if err != nil {
		return batchResult{}, err
	}
	if len(deps) == 0 {
		return batchResult{}, nil
	}

	batch, count, err := buildBatchUpdates(deps)
	if err != nil {
		return batchResult{}, err
	}

	if batch.Len() > 0 {
		if err := sendBatch(ctx, conn, batch); err != nil {
			return batchResult{}, err
		}
	}

	return batchResult{
		updated: count,
		lastID:  deps[len(deps)-1].id,
	}, nil
}

func fetchDeployments(ctx context.Context, conn *postgres.Conn, lastID string) ([]depRow, error) {
	rows, err := conn.Query(ctx, `
		SELECT d.id, d.serialized
		FROM deployments d
		WHERE d.id > $2::uuid AND EXISTS (
			SELECT 1 FROM deployments_containers dc
			WHERE dc.deployments_id = d.id
			  AND dc.image_id IS NOT NULL AND dc.image_id != ''
			  AND dc.image_name_fullname IS NOT NULL AND dc.image_name_fullname != ''
			  AND (dc.image_idv2 IS NULL OR dc.image_idv2 = '')
		)
		ORDER BY d.id
		LIMIT $1`, batchSize, lastID)
	if err != nil {
		return nil, fmt.Errorf("querying deployments: %w", err)
	}
	defer rows.Close()

	deps := make([]depRow, 0, batchSize)
	for rows.Next() {
		var d depRow
		if err := rows.Scan(&d.id, &d.serialized); err != nil {
			return nil, fmt.Errorf("scanning deployment row: %w", err)
		}
		deps = append(deps, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating deployment rows: %w", err)
	}
	return deps, nil
}

func buildBatchUpdates(deps []depRow) (*pgx.Batch, int, error) {
	batch := &pgx.Batch{}
	count := 0

	for _, dep := range deps {
		d := &storage.Deployment{}
		if err := d.UnmarshalVT(dep.serialized); err != nil {
			return nil, 0, fmt.Errorf("unmarshal deployment %s: %w", dep.id, err)
		}

		blobChanged := populateContainerImageIDV2s(d)

		if blobChanged {
			newSerialized, err := d.MarshalVT()
			if err != nil {
				return nil, 0, fmt.Errorf("marshal deployment %s: %w", dep.id, err)
			}
			batch.Queue("UPDATE deployments SET serialized = $1 WHERE id = $2",
				newSerialized, pgutils.NilOrUUID(dep.id))
		}

		for idx, container := range d.GetContainers() {
			if idv2 := container.GetImage().GetIdV2(); idv2 != "" {
				batch.Queue("UPDATE deployments_containers SET image_idv2 = $1 WHERE deployments_id = $2 AND idx = $3",
					idv2, pgutils.NilOrUUID(dep.id), idx)
			}
		}

		count++
	}

	return batch, count, nil
}

func sendBatch(ctx context.Context, conn *postgres.Conn, batch *pgx.Batch) error {
	results := conn.SendBatch(ctx, batch)
	for i := 0; i < batch.Len(); i++ {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return fmt.Errorf("batch exec statement %d: %w", i, err)
		}
	}
	return results.Close()
}

// populateContainerImageIDV2s sets IdV2 on containers that have an image ID
// and full name but no IdV2. Returns true if any container was updated.
func populateContainerImageIDV2s(deployment *storage.Deployment) bool {
	changed := false
	for _, container := range deployment.GetContainers() {
		img := container.GetImage()
		if img == nil || img.GetIdV2() != "" {
			continue
		}
		if idV2 := newImageV2ID(img.GetName().GetFullName(), img.GetId()); idV2 != "" {
			img.IdV2 = idV2
			changed = true
		}
	}
	return changed
}

func newImageV2ID(fullName, digest string) string {
	if fullName == "" || digest == "" {
		return ""
	}
	return uuid.NewV5FromNonUUIDs(fullName, digest).String()
}
