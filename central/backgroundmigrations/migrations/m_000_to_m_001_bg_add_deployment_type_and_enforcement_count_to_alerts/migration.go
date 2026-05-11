package m000tom001

import (
	"context"

	"github.com/jackc/pgx/v5"
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

func backfillOrphanedDeploymentType(ctx context.Context, db postgres.DB) error {
	totalBackfilled := 0
	lastID := ""

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		ids, deployTypes, newLastID, err := readOrphanedDeploymentTypes(ctx, db, lastID)
		if err != nil {
			return err
		}
		lastID = newLastID

		if len(ids) == 0 {
			break
		}

		if err := batchUpdateDeploymentType(ctx, db, ids, deployTypes); err != nil {
			return err
		}

		totalBackfilled += len(ids)
		log.Infof("Backfilled bg_deployment_type for %d orphaned alerts (total: %d)", len(ids), totalBackfilled)

		if len(ids) < batchSize {
			break
		}
	}

	log.Infof("Successfully backfilled bg_deployment_type for %d total orphaned alerts", totalBackfilled)
	return nil
}

func readOrphanedDeploymentTypes(ctx context.Context, db postgres.DB, lastID string) (ids []string, deployTypes []string, newLastID string, err error) {
	var rows pgx.Rows

	if lastID == "" {
		rows, err = db.Query(ctx,
			`SELECT a.id, a.serialized FROM alerts a
			WHERE a.entitytype = $1
			  AND (a.bg_deployment_type IS NULL OR a.bg_deployment_type = '')
			ORDER BY a.id LIMIT $2`,
			storage.Alert_DEPLOYMENT, batchSize)
	} else {
		rows, err = db.Query(ctx,
			`SELECT a.id, a.serialized FROM alerts a
			WHERE a.entitytype = $1
			  AND (a.bg_deployment_type IS NULL OR a.bg_deployment_type = '')
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

func batchUpdateDeploymentType(ctx context.Context, db postgres.DB, ids []string, deployTypes []string) error {
	conn, err := db.Acquire(ctx)
	if err != nil {
		return errors.Wrap(err, "acquiring connection")
	}
	defer conn.Release()

	batch := &pgx.Batch{}
	for i := range ids {
		batch.Queue("UPDATE alerts SET bg_deployment_type = $1 WHERE id = $2", deployTypes[i], ids[i])
	}

	results := conn.SendBatch(ctx, batch)
	for i := 0; i < len(ids); i++ {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return errors.Wrapf(err, "updating bg_deployment_type for alert at index %d", i)
		}
	}
	return results.Close()
}

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
		lastID = newLastID

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

func processEnforcementBatch(ctx context.Context, db postgres.DB, lastID string) (count int, newLastID string, err error) {
	var rows pgx.Rows
	if lastID == "" {
		rows, err = db.Query(ctx,
			`SELECT id, serialized FROM alerts
			WHERE enforcement_action != 0
			  AND (bg_enforcementcount IS NULL OR bg_enforcementcount = 0)
			ORDER BY id LIMIT $1`,
			batchSize)
	} else {
		rows, err = db.Query(ctx,
			`SELECT id, serialized FROM alerts
			WHERE enforcement_action != 0
			  AND (bg_enforcementcount IS NULL OR bg_enforcementcount = 0)
			  AND id > $1
			ORDER BY id LIMIT $2`,
			lastID, batchSize)
	}
	if err != nil {
		return 0, "", errors.Wrap(err, "querying enforced alerts")
	}
	defer rows.Close()

	type alertUpdate struct {
		id    string
		count int32
	}
	updates := make([]alertUpdate, 0, batchSize)
	fetched := 0

	for rows.Next() {
		fetched++
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
		updates = append(updates, alertUpdate{id: id, count: enfCount})
		newLastID = id
	}

	if err := rows.Err(); err != nil {
		return 0, "", errors.Wrap(err, "iterating enforced alert rows")
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
		batch.Queue("UPDATE alerts SET bg_enforcementcount = $1 WHERE id = $2", u.count, u.id)
	}

	results := conn.SendBatch(ctx, batch)
	for i := 0; i < len(updates); i++ {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return 0, "", errors.Wrapf(err, "updating bg_enforcementcount for alert at index %d", i)
		}
	}
	if err := results.Close(); err != nil {
		return 0, "", errors.Wrap(err, "closing batch results")
	}

	return fetched, newLastID, nil
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
