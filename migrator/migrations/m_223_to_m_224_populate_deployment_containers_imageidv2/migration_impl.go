package m223tom224

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_223_to_m_224_populate_deployment_containers_imageidv2/schema"
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

// migrate updates image_idv2 in both the deployments_containers column and the
// deployments.serialized proto blob. Updating the blob is necessary because
// the platform reprocessor reads the blob and upserts back — if the blob
// doesn't have id_v2, the reprocessor clobbers the column value.
func migrate(database *types.Databases) error {
	pgutils.CreateTableFromModel(database.DBCtx, database.GormDB, schema.CreateTableDeploymentsStmt)
	db := database.PostgresDB
	ctx := database.DBCtx

	updatedCount := 0
	var upsertErrors *multierror.Error

	// Paginate by deployment ID to guarantee forward progress even if some
	// deployments consistently fail to update.
	lastID := uuid.Nil.String()
	for {
		batchUpdated, newLastID, err := migrateBatch(ctx, db, lastID, &upsertErrors)
		if err != nil {
			return err
		}
		if newLastID == "" {
			break
		}
		lastID = newLastID
		updatedCount += batchUpdated
	}

	log.Infof("Populated image_idv2 for %d deployments", updatedCount)
	if upsertErrors.ErrorOrNil() != nil {
		log.Errorf("Errors during migration: %v", upsertErrors.ErrorOrNil())
	}
	return nil
}

type depRow struct {
	id         string
	serialized []byte
}

// migrateBatch processes one page of deployments starting after lastID.
// Returns the number of deployments updated, the last deployment ID processed
// (empty if no rows found), and any fatal error.
func migrateBatch(ctx context.Context, db postgres.DB, lastID string, upsertErrors **multierror.Error) (int, string, error) {
	rows, err := db.Query(ctx, `
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
		return 0, "", fmt.Errorf("querying deployments: %w", err)
	}
	defer rows.Close()

	var deps []depRow
	for rows.Next() {
		var d depRow
		if err := rows.Scan(&d.id, &d.serialized); err != nil {
			*upsertErrors = multierror.Append(*upsertErrors, fmt.Errorf("scanning deployment row: %w", err))
			continue
		}
		deps = append(deps, d)
	}
	if err := rows.Err(); err != nil {
		return 0, "", fmt.Errorf("iterating deployment rows: %w", err)
	}

	if len(deps) == 0 {
		return 0, "", nil
	}
	newLastID := deps[len(deps)-1].id

	conn, err := db.Acquire(ctx)
	if err != nil {
		return 0, "", fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	batch := &pgx.Batch{}
	batchDeploymentCount := 0

	for _, dep := range deps {
		d := &storage.Deployment{}
		if err := d.UnmarshalVT(dep.serialized); err != nil {
			*upsertErrors = multierror.Append(*upsertErrors, fmt.Errorf("unmarshal deployment %s: %w", dep.id, err))
			continue
		}

		blobChanged := populateContainerImageIDV2s(d)

		if blobChanged {
			newSerialized, err := d.MarshalVT()
			if err != nil {
				*upsertErrors = multierror.Append(*upsertErrors, fmt.Errorf("marshal deployment %s: %w", dep.id, err))
				continue
			}
			batch.Queue("UPDATE deployments SET serialized = $1 WHERE id = $2",
				newSerialized, pgutils.NilOrUUID(dep.id))
		}

		// Always update columns from proto values to ensure they match the blob.
		for idx, container := range d.GetContainers() {
			if idv2 := container.GetImage().GetIdV2(); idv2 != "" {
				batch.Queue("UPDATE deployments_containers SET image_idv2 = $1 WHERE deployments_id = $2 AND idx = $3",
					idv2, pgutils.NilOrUUID(dep.id), idx)
			}
		}

		batchDeploymentCount++
	}

	if batch.Len() > 0 {
		results := conn.SendBatch(ctx, batch)
		for i := 0; i < batch.Len(); i++ {
			if _, err := results.Exec(); err != nil {
				*upsertErrors = multierror.Append(*upsertErrors, fmt.Errorf("batch exec: %w", err))
			}
		}
		if err := results.Close(); err != nil {
			return 0, "", fmt.Errorf("closing batch results: %w", err)
		}
	}

	return batchDeploymentCount, newLastID, nil
}

// populateContainerImageIDV2s sets IdV2 on containers that have an image ID
// and full name but no IdV2. Returns true if any container was updated.
func populateContainerImageIDV2s(deployment *storage.Deployment) bool {
	changed := false
	for _, container := range deployment.GetContainers() {
		img := container.GetImage()
		if img.GetIdV2() != "" {
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
