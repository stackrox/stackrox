//go:build sql_integration

package m223tom224

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	updatedSchema "github.com/stackrox/rox/migrator/migrations/m_223_to_m_224_add_deployment_type_and_enforcement_count_to_alerts/schema"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_223_to_m_224_add_deployment_type_and_enforcement_count_to_alerts/test/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

type migrationTestSuite struct {
	suite.Suite

	db  *pghelper.TestPostgres
	ctx context.Context
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pghelper.ForT(s.T(), false)
}

func (s *migrationTestSuite) TestMigration() {
	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	// Create pre-migration schema (no deployment_type, no enforcementcount columns).
	pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, oldSchema.CreateTableAlertsStmt)

	// Also create a minimal deployments table for the JOIN backfill.
	_, err := dbs.PostgresDB.Exec(s.ctx,
		`CREATE TABLE IF NOT EXISTS deployments (
			id UUID PRIMARY KEY,
			type VARCHAR
		)`)
	s.Require().NoError(err)

	// Insert test deployments.
	deploymentID := fixtureconsts.Deployment1
	_, err = dbs.PostgresDB.Exec(s.ctx,
		`INSERT INTO deployments (id, type) VALUES ($1, $2)`,
		deploymentID, "DaemonSet")
	s.Require().NoError(err)

	orphanedDeploymentID := uuid.NewV4().String()

	// Generate UUIDs for each test alert and track them by name.
	alertIDs := map[string]string{
		"deploy-live":       uuid.NewV4().String(),
		"deploy-orphan":     uuid.NewV4().String(),
		"deploy-enforced":   uuid.NewV4().String(),
		"runtime-killpod":   uuid.NewV4().String(),
		"resource":          uuid.NewV4().String(),
		"resolved-enforced": uuid.NewV4().String(),
	}

	// Insert test alerts covering all migration paths.
	alerts := []*storage.Alert{
		// 1. Deployment alert with live deployment (JOIN path).
		makeAlert(alertIDs["deploy-live"], deploymentID, "Deployment", storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE, nil),
		// 2. Deployment alert with deleted deployment (orphan blob path).
		makeAlert(alertIDs["deploy-orphan"], orphanedDeploymentID, "StatefulSet", storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE, nil),
		// 3. Active DEPLOY alert with enforcement (enforcement_count = 1).
		makeAlert(alertIDs["deploy-enforced"], deploymentID, "Deployment", storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE,
			&storage.Alert_Enforcement{Action: storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT}),
		// 4. Active RUNTIME alert with KILL_POD enforcement (enforcement_count = distinct pods).
		makeRuntimeKillPodAlert(alertIDs["runtime-killpod"], deploymentID),
		// 5. Resource alert (no deployment_type needed).
		makeResourceAlert(alertIDs["resource"]),
		// 6. Resolved alert with enforcement (enforcement_count should stay 0).
		makeAlert(alertIDs["resolved-enforced"], deploymentID, "Deployment", storage.LifecycleStage_DEPLOY, storage.ViolationState_RESOLVED,
			&storage.Alert_Enforcement{Action: storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT}),
	}

	// Insert alerts using raw SQL to handle UUID NULL semantics correctly.
	for _, alert := range alerts {
		serialized, err := alert.MarshalVT()
		s.Require().NoError(err)

		_, err = dbs.PostgresDB.Exec(s.ctx,
			`INSERT INTO alerts (id, policy_id, policy_name, lifecyclestage, clusterid, clustername,
				namespace, deployment_id, deployment_name, deployment_inactive,
				enforcement_action, time, state, entitytype,
				resource_resourcetype, resource_name, serialized)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
			pgutils.NilOrUUID(alert.GetId()),
			alert.GetPolicy().GetId(),
			alert.GetPolicy().GetName(),
			alert.GetLifecycleStage(),
			pgutils.NilOrUUID(alert.GetClusterId()),
			alert.GetClusterName(),
			alert.GetNamespace(),
			pgutils.NilOrUUID(alert.GetDeployment().GetId()),
			alert.GetDeployment().GetName(),
			alert.GetDeployment().GetInactive(),
			alert.GetEnforcement().GetAction(),
			protocompat.NilOrTime(alert.GetTime()),
			alert.GetState(),
			alert.GetEntityType(),
			alert.GetResource().GetResourceType(),
			alert.GetResource().GetName(),
			serialized,
		)
		s.Require().NoError(err)
	}

	// Verify columns don't exist yet.
	var colCount int
	err = dbs.PostgresDB.QueryRow(s.ctx,
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'alerts' AND column_name = 'deployment_type'`).Scan(&colCount)
	s.Require().NoError(err)
	s.Require().Equal(0, colCount, "deployment_type column should not exist before migration")

	// Run migration.
	s.Require().NoError(migration.Run(dbs))

	// Verify deployment_type was backfilled.
	// Use COALESCE to handle NULL → empty string for scanning.
	var deployType string

	// Live deployment: should get type from JOIN.
	err = dbs.PostgresDB.QueryRow(s.ctx,
		`SELECT COALESCE(deployment_type, '') FROM alerts WHERE id = $1`, alertIDs["deploy-live"]).Scan(&deployType)
	s.Require().NoError(err)
	s.Equal("DaemonSet", deployType, "live deployment alert should get type from deployments table")

	// Orphaned deployment: should get type from blob.
	err = dbs.PostgresDB.QueryRow(s.ctx,
		`SELECT COALESCE(deployment_type, '') FROM alerts WHERE id = $1`, alertIDs["deploy-orphan"]).Scan(&deployType)
	s.Require().NoError(err)
	s.Equal("StatefulSet", deployType, "orphaned deployment alert should get type from blob")

	// Resource alert: should be NULL/empty (not a deployment).
	err = dbs.PostgresDB.QueryRow(s.ctx,
		`SELECT COALESCE(deployment_type, '') FROM alerts WHERE id = $1`, alertIDs["resource"]).Scan(&deployType)
	s.Require().NoError(err)
	s.Equal("", deployType, "resource alert should have empty deployment_type")

	// Verify enforcement_count was backfilled.
	var enfCount int32

	// DEPLOY + enforced + ACTIVE: should be 1.
	err = dbs.PostgresDB.QueryRow(s.ctx,
		`SELECT COALESCE(enforcementcount, 0) FROM alerts WHERE id = $1`, alertIDs["deploy-enforced"]).Scan(&enfCount)
	s.Require().NoError(err)
	s.Equal(int32(1), enfCount, "active deploy enforced alert should have enforcement_count=1")

	// RUNTIME + KILL_POD + ACTIVE: should be 2 (2 distinct pods).
	err = dbs.PostgresDB.QueryRow(s.ctx,
		`SELECT COALESCE(enforcementcount, 0) FROM alerts WHERE id = $1`, alertIDs["runtime-killpod"]).Scan(&enfCount)
	s.Require().NoError(err)
	s.Equal(int32(2), enfCount, "runtime kill-pod alert should have enforcement_count=2 (distinct pods)")

	// RESOLVED + enforced: should be 0 (not active).
	err = dbs.PostgresDB.QueryRow(s.ctx,
		`SELECT COALESCE(enforcementcount, 0) FROM alerts WHERE id = $1`, alertIDs["resolved-enforced"]).Scan(&enfCount)
	s.Require().NoError(err)
	s.Equal(int32(0), enfCount, "resolved alert should have enforcement_count=0")

	// No enforcement: should be 0.
	err = dbs.PostgresDB.QueryRow(s.ctx,
		`SELECT COALESCE(enforcementcount, 0) FROM alerts WHERE id = $1`, alertIDs["deploy-live"]).Scan(&enfCount)
	s.Require().NoError(err)
	s.Equal(int32(0), enfCount, "alert without enforcement should have enforcement_count=0")

	// Verify blob consistency for enforced alert: re-read and check the proto field.
	var serialized []byte
	err = dbs.PostgresDB.QueryRow(s.ctx,
		`SELECT serialized FROM alerts WHERE id = $1`, alertIDs["deploy-enforced"]).Scan(&serialized)
	s.Require().NoError(err)
	updatedAlert := &storage.Alert{}
	s.Require().NoError(updatedAlert.UnmarshalVT(serialized))
	s.Equal(int32(1), updatedAlert.GetEnforcementCount(), "serialized blob should contain enforcement_count=1")

	// Verify idempotency: run migration again.
	s.Require().NoError(migration.Run(dbs))

	// Spot check values haven't changed.
	err = dbs.PostgresDB.QueryRow(s.ctx,
		`SELECT COALESCE(deployment_type, '') FROM alerts WHERE id = $1`, alertIDs["deploy-live"]).Scan(&deployType)
	s.Require().NoError(err)
	s.Equal("DaemonSet", deployType)

	// Verify pre-migration queries still work (backwards compatibility).
	var count int
	err = dbs.PostgresDB.QueryRow(s.ctx,
		`SELECT COUNT(*) FROM alerts WHERE state = $1`, storage.ViolationState_ACTIVE).Scan(&count)
	s.Require().NoError(err)
	s.Equal(5, count, "pre-migration state query should still work")

	_ = updatedSchema.AlertsTableName // reference to satisfy import
}

func makeAlert(id, deploymentID, deploymentType string, lifecycle storage.LifecycleStage, state storage.ViolationState, enforcement *storage.Alert_Enforcement) *storage.Alert {
	a := &storage.Alert{
		Id:             id,
		LifecycleStage: lifecycle,
		State:          state,
		Time:           protocompat.TimestampNow(),
		Policy: &storage.Policy{
			Id:   uuid.NewV4().String(),
			Name: "test-policy",
		},
		ClusterId:   fixtureconsts.Cluster1,
		ClusterName: "test-cluster",
		Namespace:   "default",
		Entity: &storage.Alert_Deployment_{
			Deployment: &storage.Alert_Deployment{
				Id:   deploymentID,
				Name: "test-deployment",
				Type: deploymentType,
			},
		},
		EntityType:  storage.Alert_DEPLOYMENT,
		Enforcement: enforcement,
	}
	return a
}

func makeRuntimeKillPodAlert(id, deploymentID string) *storage.Alert {
	a := makeAlert(id, deploymentID, "Deployment", storage.LifecycleStage_RUNTIME, storage.ViolationState_ACTIVE,
		&storage.Alert_Enforcement{Action: storage.EnforcementAction_KILL_POD_ENFORCEMENT})
	a.ProcessViolation = &storage.Alert_ProcessViolation{
		Message: "kill pod",
		Processes: []*storage.ProcessIndicator{
			{Id: uuid.NewV4().String(), PodId: "pod-1", Signal: &storage.ProcessSignal{Name: "p1"}},
			{Id: uuid.NewV4().String(), PodId: "pod-2", Signal: &storage.ProcessSignal{Name: "p2"}},
			{Id: uuid.NewV4().String(), PodId: "pod-1", Signal: &storage.ProcessSignal{Name: "p3"}}, // duplicate pod
		},
	}
	return a
}

func makeResourceAlert(id string) *storage.Alert {
	return &storage.Alert{
		Id:             id,
		LifecycleStage: storage.LifecycleStage_RUNTIME,
		State:          storage.ViolationState_ACTIVE,
		Time:           protocompat.TimestampNow(),
		Policy: &storage.Policy{
			Id:   uuid.NewV4().String(),
			Name: "audit-policy",
		},
		ClusterId:   fixtureconsts.Cluster1,
		ClusterName: "test-cluster",
		Namespace:   "default",
		Entity: &storage.Alert_Resource_{
			Resource: &storage.Alert_Resource{
				ResourceType: storage.Alert_Resource_SECRETS,
				Name:         "my-secret",
			},
		},
		EntityType: storage.Alert_RESOURCE,
	}
}
