package m220tom221

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	"github.com/stackrox/rox/migrator/migrations/m_220_to_m_221_add_deployment_hash_column/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	log       = loghelper.LogWrapper{}
	batchSize = 500
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())

	// Use GORM to add the hash column to the deployments table
	pgutils.CreateTableFromModel(ctx, database.GormDB, schema.CreateTableDeploymentsStmt)

	if err := backfillHash(ctx, database.PostgresDB, schema.DeploymentsTableName); err != nil {
		log.WriteToStderrf("unable to backfill hash: %v", err)
		return err
	}

	return nil
}

func processDeploymentRows(db postgres.DB, ctx context.Context, table string, lastID string) (ids []string, hashes []uint64, newLastID string, err error) {
	var rows pgx.Rows

	if lastID == "" {
		// First iteration - no WHERE clause
		rows, err = db.Query(ctx, "SELECT id, serialized FROM "+table+" ORDER BY id LIMIT $1", batchSize)
	} else {
		// Subsequent iterations - use keyset pagination
		rows, err = db.Query(ctx, "SELECT id, serialized FROM "+table+" WHERE id > $1 ORDER BY id LIMIT $2", lastID, batchSize)
	}

	if err != nil {
		return nil, nil, "", errors.Wrap(err, "querying deployments for backfill")
	}
	defer rows.Close()

	ids = make([]string, 0, batchSize)
	hashes = make([]uint64, 0, batchSize)

	for rows.Next() {
		var id string
		var serialized []byte
		if err := rows.Scan(&id, &serialized); err != nil {
			return nil, nil, "", errors.Wrap(err, "scanning deployment row")
		}

		deployment := &storage.Deployment{}
		if err := deployment.UnmarshalVT(serialized); err != nil {
			return nil, nil, "", errors.Wrapf(err, "deserializing deployment %s", id)
		}

		ids = append(ids, id)
		hashes = append(hashes, deployment.GetHash())
		newLastID = id
	}

	if err := rows.Err(); err != nil {
		return nil, nil, "", errors.Wrapf(err, "failed to get rows for %s", table)
	}

	return ids, hashes, newLastID, nil
}

func updateDeploymentHashes(db postgres.DB, ctx context.Context, table string, ids []string, hashes []uint64) error {
	conn, err := db.Acquire(ctx)
	if err != nil {
		return errors.Wrap(err, "acquiring database connection")
	}
	defer conn.Release()

	batch := &pgx.Batch{}
	for i := range ids {
		batch.Queue("UPDATE "+table+" SET hash = $1 WHERE id = $2", hashes[i], ids[i])
	}

	results := conn.SendBatch(ctx, batch)

	for i := 0; i < len(ids); i++ {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return errors.Wrapf(err, "updating hash for deployment at index %d", i)
		}
	}

	if err := results.Close(); err != nil {
		return errors.Wrap(err, "closing batch results")
	}
	return nil
}

func backfillHash(ctx context.Context, db postgres.DB, table string) error {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultMigrationTimeout)
	defer cancel()

	totalBackfilled := 0
	lastID := ""

	for {
		ids, hashes, newLastID, err := processDeploymentRows(db, ctx, table, lastID)
		if err != nil {
			return err
		}

		lastID = newLastID

		if len(ids) == 0 {
			break
		}

		if err := updateDeploymentHashes(db, ctx, table, ids, hashes); err != nil {
			return err
		}

		totalBackfilled += len(ids)
		log.WriteToStderrf("Backfilled hash for %d deployments (total: %d)", len(ids), totalBackfilled)

		// Break if we got fewer rows than batchSize (last batch)
		if len(ids) < batchSize {
			break
		}
	}

	log.WriteToStderrf("Successfully backfilled hash for %d total deployments", totalBackfilled)
	return nil
}
