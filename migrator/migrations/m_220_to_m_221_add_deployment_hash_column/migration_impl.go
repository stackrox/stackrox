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

func backfillHash(ctx context.Context, db postgres.DB, table string) error {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultMigrationTimeout)
	defer cancel()

	totalBackfilled := 0
	lastID := ""

	for {
		var rows pgx.Rows
		var err error

		if lastID == "" {
			// First iteration - no WHERE clause
			rows, err = db.Query(ctx, "SELECT id, serialized FROM "+table+" ORDER BY id LIMIT $1", batchSize)
		} else {
			// Subsequent iterations - use keyset pagination
			rows, err = db.Query(ctx, "SELECT id, serialized FROM "+table+" WHERE id > $1 ORDER BY id LIMIT $2", lastID, batchSize)
		}

		if err != nil {
			return errors.Wrap(err, "querying deployments for backfill")
		}

		ids := make([]string, 0, batchSize)
		hashes := make([]uint64, 0, batchSize)

		for rows.Next() {
			var id string
			var serialized []byte
			if err := rows.Scan(&id, &serialized); err != nil {
				rows.Close()
				return errors.Wrap(err, "scanning deployment row")
			}

			deployment := &storage.Deployment{}
			if err := deployment.UnmarshalVT(serialized); err != nil {
				rows.Close()
				return errors.Wrapf(err, "deserializing deployment %s", id)
			}

			ids = append(ids, id)
			hashes = append(hashes, deployment.GetHash())
			lastID = id // Track last ID for next iteration
		}

		if err := rows.Err(); err != nil {
			rows.Close()
			return errors.Wrapf(rows.Err(), "failed to get rows for %s", table)
		}
		rows.Close()

		if len(ids) == 0 {
			break
		}

		// Acquire a connection to use SendBatch
		conn, err := db.Acquire(ctx)
		if err != nil {
			return errors.Wrap(err, "acquiring database connection")
		}

		// Bulk update deployments with their hash values using batch
		batch := &pgx.Batch{}
		for i := range ids {
			batch.Queue("UPDATE "+table+" SET hash = $1 WHERE id = $2", hashes[i], ids[i])
		}

		results := conn.SendBatch(ctx, batch)
		for i := 0; i < len(ids); i++ {
			if _, err := results.Exec(); err != nil {
				_ = results.Close()
				conn.Release()
				return errors.Wrapf(err, "updating hash for deployment at index %d", i)
			}
		}
		_ = results.Close()
		conn.Release()

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
