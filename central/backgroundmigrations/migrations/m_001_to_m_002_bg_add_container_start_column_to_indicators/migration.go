package m001tom002

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/backgroundmigrations/migrations"
	"github.com/stackrox/rox/central/backgroundmigrations/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/protocompat"
)

var (
	log       = logging.LoggerForModule()
	batchSize = 5000
)

func init() {
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum:     1,
		VersionAfterSeqNum: 2,
		Description:        "Backfill bg_containerstarttime column in process_indicators from serialized blob",
		Run:                run,
	})
}

func run(ctx context.Context, db postgres.DB) error {
	totalBackfilled := 0
	lastID := ""

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		count, newLastID, err := processBatch(ctx, db, lastID)
		if err != nil {
			return err
		}
		if newLastID != "" {
			lastID = newLastID
		}

		if count == 0 {
			break
		}

		totalBackfilled += count
		log.Infof("Backfilled bg_containerstarttime for %d process indicators (total: %d)", count, totalBackfilled)

		if count < batchSize {
			break
		}
	}

	log.Infof("Successfully backfilled bg_containerstarttime for %d total process indicators", totalBackfilled)
	return nil
}

// processBatch uses cursor-based pagination on the PK index with
// FOR UPDATE SKIP LOCKED inside a transaction to prevent stale reads
// from concurrent Central writes to the serialized blob.
func processBatch(ctx context.Context, db postgres.DB, lastID string) (int, string, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return 0, "", errors.Wrap(err, "beginning transaction")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var query string
	var args []interface{}
	if lastID == "" {
		query = `SELECT id, serialized FROM process_indicators
			ORDER BY id LIMIT $1
			FOR UPDATE SKIP LOCKED`
		args = []interface{}{batchSize}
	} else {
		query = `SELECT id, serialized FROM process_indicators
			WHERE id > $1
			ORDER BY id LIMIT $2
			FOR UPDATE SKIP LOCKED`
		args = []interface{}{lastID, batchSize}
	}

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return 0, "", errors.Wrap(err, "querying process indicators")
	}

	type indicatorUpdate struct {
		id             string
		containerStart *protocompat.Timestamp
	}
	updates := make([]indicatorUpdate, 0, batchSize)
	var newLastID string

	for rows.Next() {
		var id string
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			rows.Close()
			return 0, "", errors.Wrap(err, "scanning process indicator row")
		}

		indicator := &storage.ProcessIndicator{}
		if err := indicator.UnmarshalVT(data); err != nil {
			rows.Close()
			return 0, "", errors.Wrapf(err, "deserializing process indicator %s", id)
		}

		newLastID = id
		updates = append(updates, indicatorUpdate{
			id:             id,
			containerStart: indicator.GetContainerStartTime(),
		})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, "", errors.Wrap(err, "iterating process indicator rows")
	}

	if len(updates) == 0 {
		return 0, newLastID, nil
	}

	for _, u := range updates {
		t := protocompat.NilOrTime(u.containerStart)
		if _, err := tx.Exec(ctx, "UPDATE process_indicators SET bg_containerstarttime = $1 WHERE id = $2", t, u.id); err != nil {
			return 0, "", errors.Wrapf(err, "updating bg_containerstarttime for indicator %s", u.id)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, "", errors.Wrap(err, "committing process indicator batch")
	}

	return len(updates), newLastID, nil
}
