package m001tom002

import (
	"context"

	"github.com/jackc/pgx/v5"
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
		lastID = newLastID

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

func processBatch(ctx context.Context, db postgres.DB, lastID string) (count int, newLastID string, err error) {
	var rows pgx.Rows
	if lastID == "" {
		rows, err = db.Query(ctx,
			`SELECT id, serialized FROM process_indicators
			WHERE bg_containerstarttime IS NULL
			ORDER BY id LIMIT $1`,
			batchSize)
	} else {
		rows, err = db.Query(ctx,
			`SELECT id, serialized FROM process_indicators
			WHERE bg_containerstarttime IS NULL
			  AND id > $1
			ORDER BY id LIMIT $2`,
			lastID, batchSize)
	}
	if err != nil {
		return 0, "", errors.Wrap(err, "querying process indicators")
	}
	defer rows.Close()

	type indicatorUpdate struct {
		id             string
		containerStart *protocompat.Timestamp
	}
	updates := make([]indicatorUpdate, 0, batchSize)

	for rows.Next() {
		var id string
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			return 0, "", errors.Wrap(err, "scanning process indicator row")
		}

		indicator := &storage.ProcessIndicator{}
		if err := indicator.UnmarshalVT(data); err != nil {
			return 0, "", errors.Wrapf(err, "deserializing process indicator %s", id)
		}

		updates = append(updates, indicatorUpdate{
			id:             id,
			containerStart: indicator.GetContainerStartTime(),
		})
		newLastID = id
	}

	if err := rows.Err(); err != nil {
		return 0, "", errors.Wrap(err, "iterating process indicator rows")
	}

	if len(updates) == 0 {
		return 0, newLastID, nil
	}

	conn, err := db.Acquire(ctx)
	if err != nil {
		return 0, "", errors.Wrap(err, "acquiring connection")
	}
	defer conn.Release()

	batch := &pgx.Batch{}
	for _, u := range updates {
		t := protocompat.NilOrTime(u.containerStart)
		batch.Queue("UPDATE process_indicators SET bg_containerstarttime = $1 WHERE id = $2", t, u.id)
	}

	results := conn.SendBatch(ctx, batch)
	for i := 0; i < len(updates); i++ {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return 0, "", errors.Wrapf(err, "updating bg_containerstarttime for indicator at index %d", i)
		}
	}
	if err := results.Close(); err != nil {
		return 0, "", errors.Wrap(err, "closing batch results")
	}

	return len(updates), newLastID, nil
}
