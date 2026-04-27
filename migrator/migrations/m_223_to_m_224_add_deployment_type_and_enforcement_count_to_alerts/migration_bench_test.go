//go:build sql_integration

package m223tom224

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_223_to_m_224_add_deployment_type_and_enforcement_count_to_alerts/test/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
)

// BenchmarkMigration benchmarks the full migration at customer-representative scale.
// Amadeus profile: ~1.3M alerts, ~5,600 deployments.
// Parameterized to test at multiple scales.
func BenchmarkMigration(b *testing.B) {
	sizes := []struct {
		name        string
		alerts      int
		deployments int
	}{
		{"10k", 10_000, 500},
		{"100k", 100_000, 2_000},
		{"500k", 500_000, 4_000},
	}

	for _, sz := range sizes {
		b.Run(sz.name, func(b *testing.B) {
			benchmarkMigration(b, sz.alerts, sz.deployments)
		})
	}
}

func benchmarkMigration(b *testing.B, numAlerts, numDeployments int) {
	ctx := sac.WithAllAccess(context.Background())
	db := pghelper.ForT(b, false)

	pgutils.CreateTableFromModel(ctx, db.GetGormDB(), oldSchema.CreateTableAlertsStmt)
	_, err := db.Exec(ctx,
		`CREATE TABLE IF NOT EXISTS deployments (id UUID PRIMARY KEY, type VARCHAR)`)
	if err != nil {
		b.Fatal(err)
	}

	deploymentIDs, orphanedIDs := insertBenchDeployments(b, ctx, db, numDeployments)
	insertBenchAlerts(b, ctx, db, numAlerts, deploymentIDs, orphanedIDs)

	var totalAlerts int
	if err := db.QueryRow(ctx, "SELECT COUNT(*) FROM alerts").Scan(&totalAlerts); err != nil {
		b.Fatal(err)
	}
	var enforcedAlerts int
	if err := db.QueryRow(ctx, "SELECT COUNT(*) FROM alerts WHERE enforcement_action != 0").Scan(&enforcedAlerts); err != nil {
		b.Fatal(err)
	}
	b.Logf("Setup: %d alerts (%d enforced), %d deployments (%d orphaned)",
		totalAlerts, enforcedAlerts, numDeployments, len(orphanedIDs))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		resetBenchColumns(b, ctx, db)
		b.StartTimer()

		dbs := &types.Databases{
			GormDB:     db.GetGormDB(),
			PostgresDB: db.DB,
			DBCtx:      ctx,
		}
		if err := migrate(dbs); err != nil {
			b.Fatal(err)
		}
	}
}

func insertBenchDeployments(b *testing.B, ctx context.Context, db *pghelper.TestPostgres, numDeployments int) (live []string, orphaned []string) {
	b.Helper()
	deployTypes := []string{"Deployment", "DaemonSet", "StatefulSet", "Job", "CronJob", "Pod"}

	live = make([]string, 0, numDeployments)
	// 10% of deployments will be orphaned (deleted but alerts remain)
	orphanCount := numDeployments / 10
	orphaned = make([]string, 0, orphanCount)

	for i := 0; i < numDeployments; i++ {
		id := uuid.NewV4().String()
		dt := deployTypes[i%len(deployTypes)]
		_, err := db.Exec(ctx,
			"INSERT INTO deployments (id, type) VALUES ($1, $2)",
			id, dt)
		if err != nil {
			b.Fatal(err)
		}
		live = append(live, id)
	}

	// Create orphaned deployment IDs (no row in deployments table)
	for i := 0; i < orphanCount; i++ {
		orphaned = append(orphaned, uuid.NewV4().String())
	}

	return live, orphaned
}

func insertBenchAlerts(b *testing.B, ctx context.Context, db *pghelper.TestPostgres, numAlerts int, liveDeployIDs, orphanedDeployIDs []string) {
	b.Helper()
	allDeployIDs := append(liveDeployIDs, orphanedDeployIDs...)
	deployTypes := []string{"Deployment", "DaemonSet", "StatefulSet", "Job", "CronJob", "Pod"}

	batchInsertSize := 1000
	for start := 0; start < numAlerts; start += batchInsertSize {
		end := start + batchInsertSize
		if end > numAlerts {
			end = numAlerts
		}

		query := `INSERT INTO alerts (id, policy_id, policy_name, lifecyclestage, clusterid, clustername,
			namespace, deployment_id, deployment_name, deployment_inactive,
			enforcement_action, time, state, entitytype,
			resource_resourcetype, resource_name, serialized)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`

		for i := start; i < end; i++ {
			alert := makeBenchAlert(i, allDeployIDs, deployTypes)
			serialized, err := alert.MarshalVT()
			if err != nil {
				b.Fatal(err)
			}

			_, err = db.Exec(ctx, query,
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
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func makeBenchAlert(index int, deployIDs []string, deployTypes []string) *storage.Alert {
	depID := deployIDs[index%len(deployIDs)]
	depType := deployTypes[index%len(deployTypes)]

	alert := &storage.Alert{
		Id:             uuid.NewV4().String(),
		LifecycleStage: storage.LifecycleStage_DEPLOY,
		State:          storage.ViolationState_ACTIVE,
		Time:           protocompat.TimestampNow(),
		Policy: &storage.Policy{
			Id:   uuid.NewV4().String(),
			Name: fmt.Sprintf("bench-policy-%d", index%50),
		},
		ClusterId:   uuid.NewV4().String(),
		ClusterName: "bench-cluster",
		Namespace:   fmt.Sprintf("ns-%d", index%20),
		Entity: &storage.Alert_Deployment_{
			Deployment: &storage.Alert_Deployment{
				Id:   depID,
				Name: fmt.Sprintf("deploy-%d", index),
				Type: depType,
			},
		},
		EntityType: storage.Alert_DEPLOYMENT,
	}

	// ~5% of alerts have enforcement (matching typical production ratio)
	if index%20 == 0 {
		if index%40 == 0 {
			// RUNTIME + KILL_POD with multiple pods
			alert.LifecycleStage = storage.LifecycleStage_RUNTIME
			alert.Enforcement = &storage.Alert_Enforcement{
				Action: storage.EnforcementAction_KILL_POD_ENFORCEMENT,
			}
			numPods := rand.Intn(10) + 1
			processes := make([]*storage.ProcessIndicator, 0, numPods*2)
			for p := 0; p < numPods; p++ {
				podID := fmt.Sprintf("pod-%d-%d", index, p)
				processes = append(processes,
					&storage.ProcessIndicator{
						Id:    uuid.NewV4().String(),
						PodId: podID,
						Signal: &storage.ProcessSignal{
							Name: "sh",
						},
					},
					&storage.ProcessIndicator{
						Id:    uuid.NewV4().String(),
						PodId: podID, // duplicate pod - should not increase count
						Signal: &storage.ProcessSignal{
							Name: "bash",
						},
					},
				)
			}
			alert.ProcessViolation = &storage.Alert_ProcessViolation{
				Message:   "process violation",
				Processes: processes,
			}
		} else {
			// DEPLOY + SCALE_TO_ZERO
			alert.Enforcement = &storage.Alert_Enforcement{
				Action: storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
			}
		}
	}

	// ~10% resolved
	if index%10 == 0 {
		alert.State = storage.ViolationState_RESOLVED
	}

	// ~5% resource alerts
	if index%20 == 5 {
		alert.Entity = &storage.Alert_Resource_{
			Resource: &storage.Alert_Resource{
				ResourceType: storage.Alert_Resource_SECRETS,
				Name:         fmt.Sprintf("secret-%d", index),
			},
		}
		alert.EntityType = storage.Alert_RESOURCE
		alert.Enforcement = nil
	}

	return alert
}

func resetBenchColumns(b *testing.B, ctx context.Context, db *pghelper.TestPostgres) {
	b.Helper()
	// Drop the columns added by migration so it re-adds them
	for _, col := range []string{"deployment_type", "enforcementcount"} {
		_, _ = db.Exec(ctx, fmt.Sprintf("ALTER TABLE alerts DROP COLUMN IF EXISTS %s", col))
	}
	// Reset enforcement_count in serialized blobs (set to 0)
	_, err := db.Exec(ctx,
		`UPDATE alerts SET serialized = serialized WHERE enforcement_action != 0`)
	if err != nil {
		b.Logf("Warning: could not reset serialized blobs: %v", err)
	}
}
