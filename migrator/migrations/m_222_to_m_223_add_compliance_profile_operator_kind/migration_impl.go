package m222tom223

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	"github.com/stackrox/rox/migrator/migrations/m_222_to_m_223_add_compliance_profile_operator_kind/schema"
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
	ctx := sac.WithAllAccess(database.DBCtx)
	pgutils.CreateTableFromModel(ctx, database.GormDB, schema.CreateTableComplianceOperatorProfileV2Stmt)
	if err := backfillOperatorKind(ctx, database.PostgresDB); err != nil {
		log.WriteToStderrf("unable to backfill compliance profile operator kind: %v", err)
		return err
	}
	return nil
}

func processProfileRows(db postgres.DB, ctx context.Context, lastID string) (ids []string, kinds []int32, newLastID string, err error) {
	var rows pgx.Rows
	if lastID == "" {
		rows, err = db.Query(ctx, "SELECT id, serialized FROM compliance_operator_profile_v2 ORDER BY id LIMIT $1", batchSize)
	} else {
		rows, err = db.Query(ctx, "SELECT id, serialized FROM compliance_operator_profile_v2 WHERE id > $1 ORDER BY id LIMIT $2", lastID, batchSize)
	}
	if err != nil {
		return nil, nil, "", errors.Wrap(err, "querying compliance_operator_profile_v2 for operator kind backfill")
	}
	defer rows.Close()

	ids = make([]string, 0, batchSize)
	kinds = make([]int32, 0, batchSize)

	for rows.Next() {
		var id string
		var serialized []byte
		if err := rows.Scan(&id, &serialized); err != nil {
			return nil, nil, "", errors.Wrap(err, "scanning compliance_operator_profile_v2 row")
		}
		profile := &storage.ComplianceOperatorProfileV2{}
		if err := profile.UnmarshalVT(serialized); err != nil {
			return nil, nil, "", errors.Wrapf(err, "deserializing compliance profile %s", id)
		}
		ids = append(ids, id)
		kinds = append(kinds, int32(profile.GetOperatorKind()))
		newLastID = id
	}
	if err := rows.Err(); err != nil {
		return nil, nil, "", errors.Wrap(err, "iterating compliance_operator_profile_v2 rows")
	}
	return ids, kinds, newLastID, nil
}

func updateOperatorKinds(db postgres.DB, ctx context.Context, ids []string, kinds []int32) error {
	conn, err := db.Acquire(ctx)
	if err != nil {
		return errors.Wrap(err, "acquiring database connection")
	}
	defer conn.Release()

	batch := &pgx.Batch{}
	for i := range ids {
		batch.Queue("UPDATE compliance_operator_profile_v2 SET operatorkind = $1 WHERE id = $2", kinds[i], ids[i])
	}

	results := conn.SendBatch(ctx, batch)
	for i := 0; i < len(ids); i++ {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return errors.Wrapf(err, "updating operatorkind for profile at index %d", i)
		}
	}
	return errors.Wrap(results.Close(), "closing batch results")
}

func backfillOperatorKind(ctx context.Context, db postgres.DB) error {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultMigrationTimeout)
	defer cancel()

	total := 0
	lastID := ""
	for {
		ids, kinds, newLastID, err := processProfileRows(db, ctx, lastID)
		if err != nil {
			return err
		}
		lastID = newLastID
		if len(ids) == 0 {
			break
		}
		if err := updateOperatorKinds(db, ctx, ids, kinds); err != nil {
			return err
		}
		total += len(ids)
		log.WriteToStderrf("Backfilled operator kind for %d compliance profiles (total: %d)", len(ids), total)
		if len(ids) < batchSize {
			break
		}
	}
	log.WriteToStderrf("Successfully backfilled operator kind for %d compliance profiles", total)
	return nil
}
