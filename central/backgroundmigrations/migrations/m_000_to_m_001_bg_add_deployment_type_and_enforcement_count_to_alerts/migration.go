package m000tom001

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/backgroundmigrations/migrations"
	"github.com/stackrox/rox/central/backgroundmigrations/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/set"
)

var (
	log       = logging.LoggerForModule()
	batchSize = 500
)

func init() {
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum:     0,
		VersionAfterSeqNum: 1,
		Description:        "Backfill bg_deployment_type and bg_enforcementcount columns in alerts",
		Run:                run,
	})
}

func run(ctx context.Context, db postgres.DB) error {
	if err := backfillDeploymentType(ctx, db); err != nil {
		return errors.Wrap(err, "backfilling bg_deployment_type")
	}

	if err := backfillOrphanedDeploymentType(ctx, db); err != nil {
		return errors.Wrap(err, "backfilling orphaned bg_deployment_type")
	}

	if err := backfillEnforcementCount(ctx, db); err != nil {
		return errors.Wrap(err, "backfilling bg_enforcementcount")
	}

	return nil
}

// backfillDeploymentType is a single atomic UPDATE via JOIN — no read-then-write,
// so no concurrency issue with Central's writes.
func backfillDeploymentType(ctx context.Context, db postgres.DB) error {
	result, err := db.Exec(ctx,
		`UPDATE alerts a SET bg_deployment_type = d.type
		 FROM deployments d
		 WHERE a.deployment_id = d.id
		   AND (a.bg_deployment_type IS NULL OR a.bg_deployment_type = '')
		   AND a.entitytype = $1`,
		storage.Alert_DEPLOYMENT,
	)
	if err != nil {
		return errors.Wrap(err, "updating bg_deployment_type via JOIN")
	}
	log.Infof("Backfilled bg_deployment_type for %d alerts via JOIN", result.RowsAffected())
	return nil
}

// backfillOrphanedDeploymentType reads the deployment type from the serialized
// blob for alerts whose deployment no longer exists in the deployments table.
// Uses cursor-based pagination on the PK index with FOR UPDATE SKIP LOCKED
// inside a transaction to avoid stale reads from concurrent Central writes.
func backfillOrphanedDeploymentType(ctx context.Context, db postgres.DB) error {
	totalBackfilled := 0
	lastID := ""

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		count, newLastID, err := processOrphanedDeploymentTypeBatch(ctx, db, lastID)
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
		log.Infof("Backfilled bg_deployment_type for %d orphaned alerts (total: %d)", count, totalBackfilled)

		if count < batchSize {
			break
		}
	}

	log.Infof("Successfully backfilled bg_deployment_type for %d total orphaned alerts", totalBackfilled)
	return nil
}

func processOrphanedDeploymentTypeBatch(ctx context.Context, db postgres.DB, lastID string) (int, string, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return 0, "", errors.Wrap(err, "beginning transaction")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var query string
	var args []interface{}
	if lastID == "" {
		query = `SELECT a.id, a.serialized FROM alerts a
			WHERE a.entitytype = $1
			ORDER BY a.id LIMIT $2
			FOR UPDATE SKIP LOCKED`
		args = []interface{}{storage.Alert_DEPLOYMENT, batchSize}
	} else {
		query = `SELECT a.id, a.serialized FROM alerts a
			WHERE a.entitytype = $1
			  AND a.id > $2
			ORDER BY a.id LIMIT $3
			FOR UPDATE SKIP LOCKED`
		args = []interface{}{storage.Alert_DEPLOYMENT, lastID, batchSize}
	}

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return 0, "", errors.Wrap(err, "querying orphaned deployment alerts")
	}

	type update struct {
		id         string
		deployType string
	}
	updates := make([]update, 0, batchSize)
	var newLastID string

	for rows.Next() {
		var id string
		var serialized []byte
		if err := rows.Scan(&id, &serialized); err != nil {
			rows.Close()
			return 0, "", errors.Wrap(err, "scanning orphaned alert row")
		}

		alert := &storage.Alert{}
		if err := alert.UnmarshalVT(serialized); err != nil {
			rows.Close()
			return 0, "", errors.Wrapf(err, "deserializing alert %s", id)
		}

		newLastID = id
		if alert.GetDeployment().GetType() == "" {
			continue
		}
		updates = append(updates, update{id: id, deployType: alert.GetDeployment().GetType()})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, "", errors.Wrap(err, "iterating orphaned alert rows")
	}

	for _, u := range updates {
		if _, err := tx.Exec(ctx, "UPDATE alerts SET bg_deployment_type = $1 WHERE id = $2", u.deployType, u.id); err != nil {
			return 0, "", errors.Wrapf(err, "updating bg_deployment_type for alert %s", u.id)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, "", errors.Wrap(err, "committing orphaned deployment type batch")
	}

	return len(updates), newLastID, nil
}

// backfillEnforcementCount computes enforcement_count from the serialized blob
// and writes to bg_enforcementcount. Uses cursor-based pagination on the PK
// index with FOR UPDATE SKIP LOCKED to avoid stale reads.
func backfillEnforcementCount(ctx context.Context, db postgres.DB) error {
	totalBackfilled := 0
	lastID := ""

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		count, newLastID, err := processEnforcementBatch(ctx, db, lastID)
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
		log.Infof("Backfilled bg_enforcementcount for %d alerts (total: %d)", count, totalBackfilled)

		if count < batchSize {
			break
		}
	}

	log.Infof("Successfully backfilled bg_enforcementcount for %d total alerts", totalBackfilled)
	return nil
}

func processEnforcementBatch(ctx context.Context, db postgres.DB, lastID string) (int, string, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return 0, "", errors.Wrap(err, "beginning transaction")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var query string
	var args []interface{}
	if lastID == "" {
		query = `SELECT id, serialized FROM alerts
			WHERE enforcement_action != 0
			ORDER BY id LIMIT $1
			FOR UPDATE SKIP LOCKED`
		args = []interface{}{batchSize}
	} else {
		query = `SELECT id, serialized FROM alerts
			WHERE enforcement_action != 0
			  AND id > $1
			ORDER BY id LIMIT $2
			FOR UPDATE SKIP LOCKED`
		args = []interface{}{lastID, batchSize}
	}

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return 0, "", errors.Wrap(err, "querying enforced alerts")
	}

	type alertUpdate struct {
		id    string
		count int32
	}
	updates := make([]alertUpdate, 0, batchSize)
	var newLastID string

	for rows.Next() {
		var id string
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			rows.Close()
			return 0, "", errors.Wrap(err, "scanning enforced alert row")
		}

		alert := &storage.Alert{}
		if err := alert.UnmarshalVT(data); err != nil {
			rows.Close()
			return 0, "", errors.Wrapf(err, "deserializing alert %s", id)
		}

		newLastID = id
		updates = append(updates, alertUpdate{id: id, count: computeEnforcementCount(alert)})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, "", errors.Wrap(err, "iterating enforced alert rows")
	}

	if len(updates) == 0 {
		return 0, newLastID, nil
	}

	for _, u := range updates {
		if _, err := tx.Exec(ctx, "UPDATE alerts SET bg_enforcementcount = $1 WHERE id = $2", u.count, u.id); err != nil {
			return 0, "", errors.Wrapf(err, "updating bg_enforcementcount for alert %s", u.id)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, "", errors.Wrap(err, "committing enforcement count batch")
	}

	return len(updates), newLastID, nil
}

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
