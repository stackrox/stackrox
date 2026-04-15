package m223tom224

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	"github.com/stackrox/rox/migrator/migrations/m_223_to_m_224_add_deployment_type_and_enforcement_count_to_alerts/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/alert/convert"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
)

var (
	log       = loghelper.LogWrapper{}
	batchSize = 500
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())

	// Step 1: Add new columns via GORM (idempotent — no-op if columns already exist).
	pgutils.CreateTableFromModel(ctx, database.GormDB, schema.CreateTableAlertsStmt)

	// Step 2: Backfill deployment_type from the deployments table via JOIN.
	if err := backfillDeploymentType(ctx, database.PostgresDB); err != nil {
		return errors.Wrap(err, "backfilling deployment_type")
	}

	// Step 3: Backfill deployment_type for orphaned alerts (deployment deleted)
	// by reading the value from the serialized blob.
	if err := backfillOrphanedDeploymentType(ctx, database.PostgresDB); err != nil {
		return errors.Wrap(err, "backfilling orphaned deployment_type")
	}

	// Step 4: Backfill enforcement_count for active alerts with enforcement.
	// This requires deserializing the blob to compute the count, then
	// re-serializing to keep columns and blob consistent.
	if err := backfillEnforcementCount(ctx, database.PostgresDB); err != nil {
		return errors.Wrap(err, "backfilling enforcement_count")
	}

	return nil
}

// backfillDeploymentType sets deployment_type on alert rows by JOINing with the
// deployments table. This is pure SQL — no blob deserialization needed.
func backfillDeploymentType(ctx context.Context, db postgres.DB) error {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultMigrationTimeout)
	defer cancel()

	result, err := db.Exec(ctx,
		`UPDATE alerts a SET deployment_type = d.type
		 FROM deployments d
		 WHERE a.deployment_id = d.id
		   AND (a.deployment_type IS NULL OR a.deployment_type = '')
		   AND a.entitytype = $1`,
		storage.Alert_DEPLOYMENT,
	)
	if err != nil {
		return errors.Wrap(err, "updating deployment_type via JOIN")
	}
	log.WriteToStderrf("Backfilled deployment_type for %d alerts via JOIN", result.RowsAffected())
	return nil
}

// backfillOrphanedDeploymentType handles alerts whose deployment has been deleted.
// These must be read from the serialized blob since the deployments table no longer
// has the record.
func backfillOrphanedDeploymentType(ctx context.Context, db postgres.DB) error {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultMigrationTimeout)
	defer cancel()

	totalBackfilled := 0
	lastID := ""

	for {
		ids, deployTypes, newLastID, err := readOrphanedDeploymentTypes(db, ctx, lastID)
		if err != nil {
			return err
		}
		lastID = newLastID

		if len(ids) == 0 {
			break
		}

		if err := batchUpdateDeploymentType(db, ctx, ids, deployTypes); err != nil {
			return err
		}

		totalBackfilled += len(ids)
		log.WriteToStderrf("Backfilled deployment_type for %d orphaned alerts (total: %d)", len(ids), totalBackfilled)

		if len(ids) < batchSize {
			break
		}
	}

	log.WriteToStderrf("Successfully backfilled deployment_type for %d total orphaned alerts", totalBackfilled)
	return nil
}

func readOrphanedDeploymentTypes(db postgres.DB, ctx context.Context, lastID string) (ids []string, deployTypes []string, newLastID string, err error) {
	var rows pgx.Rows

	if lastID == "" {
		rows, err = db.Query(ctx,
			`SELECT a.id, a.serialized FROM alerts a
			WHERE a.entitytype = $1
			  AND (a.deployment_type IS NULL OR a.deployment_type = '')
			ORDER BY a.id LIMIT $2`,
			storage.Alert_DEPLOYMENT, batchSize)
	} else {
		rows, err = db.Query(ctx,
			`SELECT a.id, a.serialized FROM alerts a
			WHERE a.entitytype = $1
			  AND (a.deployment_type IS NULL OR a.deployment_type = '')
			  AND a.id > $2
			ORDER BY a.id LIMIT $3`,
			storage.Alert_DEPLOYMENT, lastID, batchSize)
	}
	if err != nil {
		return nil, nil, "", errors.Wrap(err, "querying orphaned deployment alerts")
	}
	defer rows.Close()

	ids = make([]string, 0, batchSize)
	deployTypes = make([]string, 0, batchSize)

	for rows.Next() {
		var id string
		var serialized []byte
		if err := rows.Scan(&id, &serialized); err != nil {
			return nil, nil, "", errors.Wrap(err, "scanning orphaned alert row")
		}

		alert := &storage.Alert{}
		if err := alert.UnmarshalVT(serialized); err != nil {
			return nil, nil, "", errors.Wrapf(err, "deserializing alert %s", id)
		}

		ids = append(ids, id)
		deployTypes = append(deployTypes, alert.GetDeployment().GetType())
		newLastID = id
	}

	if err := rows.Err(); err != nil {
		return nil, nil, "", errors.Wrap(err, "iterating orphaned alert rows")
	}

	return ids, deployTypes, newLastID, nil
}

func batchUpdateDeploymentType(db postgres.DB, ctx context.Context, ids []string, deployTypes []string) error {
	conn, err := db.Acquire(ctx)
	if err != nil {
		return errors.Wrap(err, "acquiring connection")
	}
	defer conn.Release()

	batch := &pgx.Batch{}
	for i := range ids {
		batch.Queue("UPDATE alerts SET deployment_type = $1 WHERE id = $2", deployTypes[i], ids[i])
	}

	results := conn.SendBatch(ctx, batch)
	for i := 0; i < len(ids); i++ {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return errors.Wrapf(err, "updating deployment_type for alert at index %d", i)
		}
	}
	return results.Close()
}

// backfillEnforcementCount computes and stores enforcement_count for active
// alerts with enforcement. This is a small subset of the total alert population.
// Both the column and the serialized blob are updated to maintain consistency.
func backfillEnforcementCount(ctx context.Context, db postgres.DB) error {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultMigrationTimeout)
	defer cancel()

	totalBackfilled := 0
	lastID := ""

	for {
		count, newLastID, err := processEnforcementBatch(db, ctx, lastID)
		if err != nil {
			return err
		}
		lastID = newLastID

		if count == 0 {
			break
		}

		totalBackfilled += count
		log.WriteToStderrf("Backfilled enforcement_count for %d alerts (total: %d)", count, totalBackfilled)

		if count < batchSize {
			break
		}
	}

	log.WriteToStderrf("Successfully backfilled enforcement_count for %d total alerts", totalBackfilled)
	return nil
}

func processEnforcementBatch(db postgres.DB, ctx context.Context, lastID string) (count int, newLastID string, err error) {
	// Only active alerts with enforcement can have enforcement_count > 0.
	var rows pgx.Rows
	if lastID == "" {
		rows, err = db.Query(ctx,
			`SELECT id, serialized FROM alerts
			WHERE state = $1
			  AND enforcement_action != 0
			ORDER BY id LIMIT $2`,
			storage.ViolationState_ACTIVE, batchSize)
	} else {
		rows, err = db.Query(ctx,
			`SELECT id, serialized FROM alerts
			WHERE state = $1
			  AND enforcement_action != 0
			  AND id > $2
			ORDER BY id LIMIT $3`,
			storage.ViolationState_ACTIVE, lastID, batchSize)
	}
	if err != nil {
		return 0, "", errors.Wrap(err, "querying enforced alerts")
	}
	defer rows.Close()

	type alertUpdate struct {
		id         string
		count      int32
		serialized []byte
	}
	updates := make([]alertUpdate, 0, batchSize)

	for rows.Next() {
		var id string
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			return 0, "", errors.Wrap(err, "scanning enforced alert row")
		}

		alert := &storage.Alert{}
		if err := alert.UnmarshalVT(data); err != nil {
			return 0, "", errors.Wrapf(err, "deserializing alert %s", id)
		}

		enfCount := computeEnforcementCount(alert)
		if enfCount == 0 {
			newLastID = id
			continue
		}

		// Set on the proto and re-serialize to keep blob consistent.
		alert.EnforcementCount = enfCount
		newData, err := alert.MarshalVT()
		if err != nil {
			return 0, "", errors.Wrapf(err, "re-serializing alert %s", id)
		}

		updates = append(updates, alertUpdate{id: id, count: enfCount, serialized: newData})
		newLastID = id
	}

	if err := rows.Err(); err != nil {
		return 0, "", errors.Wrap(err, "iterating enforced alert rows")
	}

	if len(updates) == 0 {
		return 0, newLastID, nil
	}

	// Batch update both column and serialized blob.
	conn, err := db.Acquire(ctx)
	if err != nil {
		return 0, "", errors.Wrap(err, "acquiring connection")
	}
	defer conn.Release()

	batch := &pgx.Batch{}
	for _, u := range updates {
		batch.Queue("UPDATE alerts SET enforcementcount = $1, serialized = $2 WHERE id = $3",
			u.count, u.serialized, u.id)
	}

	results := conn.SendBatch(ctx, batch)
	for i := 0; i < len(updates); i++ {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return 0, "", errors.Wrapf(err, "updating enforcement_count for alert at index %d", i)
		}
	}
	if err := results.Close(); err != nil {
		return 0, "", errors.Wrap(err, "closing batch results")
	}

	return len(updates), newLastID, nil
}

// computeEnforcementCount mirrors convert.EnforcementCount but is defined here
// to keep the migration self-contained. The logic must stay in sync with the
// central runtime computation.
func computeEnforcementCount(alert *storage.Alert) int32 {
	if alert.GetEnforcement() == nil {
		return 0
	}
	if alert.GetLifecycleStage() == storage.LifecycleStage_RUNTIME {
		if alert.GetEnforcement().GetAction() != storage.EnforcementAction_KILL_POD_ENFORCEMENT {
			return 1
		}
		podIDs := set.NewStringSet()
		for _, pi := range alert.GetProcessViolation().GetProcesses() {
			podIDs.Add(pi.GetPodId())
		}
		return int32(podIDs.Cardinality())
	}
	if alert.GetLifecycleStage() == storage.LifecycleStage_DEPLOY {
		return 1
	}
	return 0
}

// Ensure computeEnforcementCount stays in sync with the canonical implementation.
var _ = convert.EnforcementCount
