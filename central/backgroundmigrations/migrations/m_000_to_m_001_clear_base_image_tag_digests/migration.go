package m000tom001

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/central/backgroundmigrations/migrations"
	"github.com/stackrox/rox/central/backgroundmigrations/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

const zeroTagID = "00000000-0000-0000-0000-000000000000"

var (
	log       = logging.LoggerForModule()
	batchSize = 500

	// afterBatchForTest runs after each committed batch. Tests use it to cancel mid-run.
	afterBatchForTest func()
)

func init() {
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum:     0,
		VersionAfterSeqNum: 1,
		Description:        "Clear base image tag digests that have no history so the next scan refetches it",
		Run:                run,
	})
}

// digestNeedsClear reports whether this tag was written before history existed.
// A tag the watcher has already refetched has history, and a crash restarts this
// migration from the first row, so those digests stay in place.
func digestNeedsClear(tag *storage.BaseImageTag) bool {
	return tag.GetManifestDigest() != "" && len(tag.GetHistory()) == 0
}

func run(ctx context.Context, db postgres.DB) error {
	totalUpdated := 0
	lastID := zeroTagID
	for {
		if err := ctx.Err(); err != nil {
			log.Infof("shutdown requested after %d base image tags, will restart on next start", totalUpdated)
			return err
		}

		updated, newLastID, err := migrateBatch(ctx, db, lastID)
		if err != nil {
			return fmt.Errorf("clearing base image tag digests after id %s: %w", lastID, err)
		}
		if newLastID == "" {
			break
		}
		lastID = newLastID
		totalUpdated += updated
		if afterBatchForTest != nil {
			afterBatchForTest()
		}
	}
	log.Infof("Cleared manifest digest for %d base image tags to force history refetch", totalUpdated)
	return nil
}

type tagRow struct {
	id         string
	serialized []byte
}

// migrateBatch locks one page of tags and clears digests that still have no history.
// The returned id is the last row read, including rows that did not need a write,
// so the next page continues past them. An empty id means the table is exhausted.
func migrateBatch(ctx context.Context, db postgres.DB, lastID string) (int, string, error) {
	conn, err := db.Acquire(ctx)
	if err != nil {
		return 0, "", fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	tx, txCtx, err := conn.Begin(ctx)
	if err != nil {
		return 0, "", fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(txCtx) }()

	rows, err := fetchBatch(txCtx, tx, lastID)
	if err != nil {
		return 0, "", err
	}
	if len(rows) == 0 {
		return 0, "", nil
	}

	updated := 0
	for _, row := range rows {
		changed, err := clearedSerialized(row)
		if err != nil {
			return 0, "", err
		}
		if changed == nil {
			continue
		}
		if _, err := tx.Exec(txCtx, `UPDATE base_image_tags SET serialized = $1 WHERE id = $2`, changed, row.id); err != nil {
			return 0, "", fmt.Errorf("clearing manifest digest for base image tag %s: %w", row.id, err)
		}
		updated++
	}

	if err := tx.Commit(txCtx); err != nil {
		return 0, "", fmt.Errorf("commit: %w", err)
	}
	return updated, rows[len(rows)-1].id, nil
}

func fetchBatch(ctx context.Context, tx *postgres.Tx, lastID string) ([]tagRow, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, serialized
		FROM base_image_tags
		WHERE id > $1::uuid
		ORDER BY id
		LIMIT $2
		FOR UPDATE SKIP LOCKED`, lastID, batchSize)
	if err != nil {
		return nil, fmt.Errorf("querying base image tags: %w", err)
	}
	defer rows.Close()

	batch := make([]tagRow, 0, batchSize)
	for rows.Next() {
		var row tagRow
		if err := rows.Scan(&row.id, &row.serialized); err != nil {
			return nil, fmt.Errorf("scanning base image tag: %w", err)
		}
		batch = append(batch, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating base image tags: %w", err)
	}
	return batch, nil
}

// clearedSerialized returns the blob with manifest_digest removed.
// A nil slice means the locked row should be left as it is.
func clearedSerialized(row tagRow) ([]byte, error) {
	tag := &storage.BaseImageTag{}
	if len(row.serialized) > 0 {
		if err := tag.UnmarshalVT(row.serialized); err != nil {
			return nil, fmt.Errorf("unmarshalling base image tag %s: %w", row.id, err)
		}
	}
	if !digestNeedsClear(tag) {
		return nil, nil
	}
	tag.ManifestDigest = ""
	updated, err := tag.MarshalVT()
	if err != nil {
		return nil, fmt.Errorf("marshalling base image tag %s: %w", row.id, err)
	}
	return updated, nil
}
