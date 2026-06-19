{{- define "TODO"}}TODO(do{{- /**/ -}}nt-merge){{end -}}
package {{.packageName}}

import (
	"context"

	"github.com/stackrox/rox/pkg/postgres"
)

// {{template "TODO"}}: Background migration checklist (see BACKGROUND_MIGRATIONS.md for full guide and examples):
//
// Concurrency & correctness:
//  - [ ] Migration is idempotent (safe to re-run after rollback or crash)
//  - [ ] ctx.Done() is checked between batches for graceful shutdown
//  - [ ] Iterates by primary key (id > $1 ORDER BY id) or by where clause on an indexed column for efficient pagination
//  - [ ] Uses FOR UPDATE SKIP LOCKED to handle concurrent Central writes
//  - [ ] Only updates rows where the column value differs from the source of truth
//
// Schema & data isolation:
//  - [ ] Never imports schemas from pkg/postgres/schema (those evolve with the latest release)
//  - [ ] If table schema is needed, freeze it inside this migration package
//  - [ ] Uses raw SQL or frozen GORM models with explicit column selection (no SELECT *)
//
// Application integration:
//  - [ ] Application code already populates the new column on INSERT/UPDATE
//  - [ ] Application code tolerates partial migration state (some rows migrated, some not)
//  - [ ] No feature flag dependencies in migration code
//
// Testing:
//  - [ ] Tests cover correctness, idempotency, existing data, and graceful shutdown

func run(ctx context.Context, db postgres.DB) error {
	_ = db  // {{template "TODO"}}: remove this line
	_ = ctx // {{template "TODO"}}: remove this line

	// {{template "TODO"}}: Migration code goes here.
	// See central/backgroundmigrations/BACKGROUND_MIGRATIONS.md for:
	//  - Example 1: Backfill a column from a serialized proto (batch read + pgx.Batch write)
	//  - Example 2: Batched SQL JOIN backfill (CTE with FOR UPDATE SKIP LOCKED)

	return nil
}
