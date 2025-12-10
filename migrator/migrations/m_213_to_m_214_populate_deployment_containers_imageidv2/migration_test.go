//go:build sql_integration

package m213tom214

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/migrator/migrations/m_213_to_m_214_populate_deployment_containers_imageidv2/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

type migrationTestSuite struct {
	suite.Suite

	db         *pghelper.TestPostgres
	ctx        context.Context
	existingDB bool
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pghelper.ForT(s.T(), false)
	// Use the below lines to use a large existing database for testing.
	// This is beneficial to test large batches at once.
	// s.db = pghelper.ForTExistingDB(s.T(), false, "7593dc135f89446b_oIIuR")
	// s.existingDB = true
	if !s.existingDB {
		pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTableDeploymentsStmt)
	}
}

func (s *migrationTestSuite) TestMigration() {
	// Test with multiple batch sizes to catch edge cases where the number of rows in the DB is on the threshold of the
	// batch size
	for _, i := range []int{3, 4, 5} {
		s.Run(fmt.Sprintf("Batch size of %d", i), func() {
			batchSize = i
			if !s.existingDB {
				deployments := map[string]*schema.DeploymentsContainers{
					"08b69e6e-a96e-5b9a-b814-93d004d01cd8": {
						ImageNameFullName: "us-central1-artifactregistry.gcr.io/gke-release/gke-release/gke-metrics-collector:20250508_2300_RC0@sha256:d074c77bdc0ee1c4245113e62d93ef1ed6f1a51960ea854a972861a6a0c774ce",
						ImageID:           "sha256:d074c77bdc0ee1c4245113e62d93ef1ed6f1a51960ea854a972861a6a0c774ce",
						ImageIDV2:         "08b69e6e-a96e-5b9a-b814-93d004d01cd8",
						DeploymentsID:     fixtures.GetDeployment().GetId(),
						Idx:               0,
					},
					"f5e05ef2-f2a8-50b2-90ac-c1a445767e94": {
						ImageNameFullName: "us-central1-artifactregistry.gcr.io/gke-release/gke-release/gke-metrics-agent:1.15.6-gke.0@sha256:8d3f6c749a8589ac729c66564b41e8babb35c5f181e774cd586c9d2761beeb96",
						ImageID:           "sha256:8d3f6c749a8589ac729c66564b41e8babb35c5f181e774cd586c9d2761beeb96",
						ImageIDV2:         "f5e05ef2-f2a8-50b2-90ac-c1a445767e94",
						DeploymentsID:     fixtures.GetDeployment().GetId(),
						Idx:               1,
					},
					"e338a8ed-8b3e-5294-8b54-fb907774e2e8": {
						ImageNameFullName: "us-central1-artifactregistry.gcr.io/gke-release/gke-release/cpvpa:v0.8.9-gke.11@sha256:ac9bb16bbfeefd9947ceb049c30fe3e6f2c18cbafc2fb213ef3ef88f940d4a29",
						ImageID:           "sha256:ac9bb16bbfeefd9947ceb049c30fe3e6f2c18cbafc2fb213ef3ef88f940d4a29",
						ImageIDV2:         "e338a8ed-8b3e-5294-8b54-fb907774e2e8",
						DeploymentsID:     fixtures.GetDeployment().GetId(),
						Idx:               2,
					},
					"6933b4da-6607-517a-aa4d-9f2ac5caac78": {
						ImageNameFullName: "us-central1-artifactregistry.gcr.io/gke-release/gke-release/fluent-bit:v1.8.1200-gke.14@sha256:fe028dfcf00bdaded6770720de8df8f3d24e841f41a968138ae00d699003aa0f",
						ImageID:           "sha256:fe028dfcf00bdaded6770720de8df8f3d24e841f41a968138ae00d699003aa0f",
						ImageIDV2:         "6933b4da-6607-517a-aa4d-9f2ac5caac78",
						DeploymentsID:     fixtures.GetDeployment().GetId(),
						Idx:               3,
					},
				}

				err := insertIntoDeployments(s.ctx, s.db, &schema.Deployments{
					ID:                            fixtures.GetDeployment().GetId(),
					Name:                          fixtures.GetDeployment().GetName(),
					Type:                          fixtures.GetDeployment().GetType(),
					Namespace:                     fixtures.GetDeployment().GetNamespace(),
					NamespaceID:                   fixtures.GetDeployment().GetNamespaceId(),
					OrchestratorComponent:         fixtures.GetDeployment().GetOrchestratorComponent(),
					Labels:                        fixtures.GetDeployment().GetLabels(),
					PodLabels:                     fixtures.GetDeployment().GetPodLabels(),
					ClusterID:                     fixtures.GetDeployment().GetClusterId(),
					ClusterName:                   fixtures.GetDeployment().GetClusterName(),
					Annotations:                   fixtures.GetDeployment().GetAnnotations(),
					Priority:                      fixtures.GetDeployment().GetPriority(),
					ServiceAccount:                fixtures.GetDeployment().GetServiceAccount(),
					ServiceAccountPermissionLevel: fixtures.GetDeployment().GetServiceAccountPermissionLevel(),
					RiskScore:                     fixtures.GetDeployment().GetRiskScore(),
					PlatformComponent:             fixtures.GetDeployment().GetPlatformComponent(),
				})
				s.Require().NoError(err)

				for _, deployment := range deployments {
					sql := "INSERT INTO deployments_containers (image_name_fullname, image_id, deployments_id, idx) VALUES ($1, $2, $3, $4)"
					_, err := s.db.Exec(s.ctx, sql, deployment.ImageNameFullName, deployment.ImageID, deployment.DeploymentsID, deployment.Idx)
					s.Require().NoError(err)
				}

				dbs := &types.Databases{
					GormDB:     s.db.GetGormDB(),
					PostgresDB: s.db.DB,
					DBCtx:      s.ctx,
				}

				s.Require().NoError(migration.Run(dbs))

				sql := "SELECT image_name_fullname, image_id, image_idv2, deployments_id, idx FROM deployments_containers"
				rows, err := s.db.Query(s.ctx, sql)
				s.Require().NoError(err)
				defer rows.Close()
				containers, err := readRowsWithIDV2(rows)
				s.Require().NoError(err)
				s.Require().Len(containers, 4)
				for _, container := range containers {
					expectedDeployment, found := deployments[container.ImageIDV2]
					s.Require().True(found)
					s.Equal(expectedDeployment, container)
				}
			} else {
				limit := 10000
				page := 0
				for {
					sql := "SELECT image_name_fullname, image_id, image_idv2, deployments_id, idx FROM deployments_containers LIMIT $1 OFFSET $2"
					rows, err := s.db.Query(s.ctx, sql, limit, page*limit)
					s.Require().NoError(err)
					containers, err := readRowsWithIDV2(rows)
					s.Require().NoError(err)
					for _, container := range containers {
						s.Equal(uuid.NewV5FromNonUUIDs(container.ImageNameFullName, container.ImageID).String(), container.ImageIDV2)
					}
					rows.Close()
					if len(containers) != limit {
						break
					}
					page++
				}
			}
			_, err := s.db.Exec(s.ctx, "DELETE FROM deployments_containers WHERE true")
			s.Require().NoError(err)
			_, err = s.db.Exec(s.ctx, "DELETE FROM deployments WHERE true")
			s.Require().NoError(err)
		})
	}
}

func readRowsWithIDV2(rows *postgres.Rows) ([]*schema.DeploymentsContainers, error) {
	var containers []*schema.DeploymentsContainers

	for rows.Next() {
		var imageName string
		var imageId string
		var imageIdV2 string
		var deploymentsID string
		var idx int

		if err := rows.Scan(&imageName, &imageId, &imageIdV2, &deploymentsID, &idx); err != nil {
			return nil, pgutils.ErrNilIfNoRows(err)
		}

		container := &schema.DeploymentsContainers{
			ImageID:           imageId,
			ImageNameFullName: imageName,
			ImageIDV2:         imageIdV2,
			DeploymentsID:     deploymentsID,
			Idx:               idx,
		}
		containers = append(containers, container)
	}

	return containers, rows.Err()
}

func insertIntoDeployments(ctx context.Context, db postgres.DB, obj *schema.Deployments) error {
	values := []interface{}{
		pgutils.NilOrUUID(obj.ID),
		obj.Name,
		obj.Type,
		obj.Namespace,
		pgutils.NilOrUUID(obj.NamespaceID),
		obj.OrchestratorComponent,
		pgutils.EmptyOrMap(obj.Labels),
		pgutils.EmptyOrMap(obj.PodLabels),
		pgutils.NilOrUUID(obj.ClusterID),
		obj.ClusterName,
		pgutils.EmptyOrMap(obj.Annotations),
		obj.Priority,
		obj.ServiceAccount,
		obj.ServiceAccountPermissionLevel,
		obj.RiskScore,
		obj.PlatformComponent,
	}

	finalStr := "INSERT INTO deployments (Id, Name, Type, Namespace, NamespaceId, OrchestratorComponent, Labels, PodLabels, ClusterId, ClusterName, Annotations, Priority, ServiceAccount, ServiceAccountPermissionLevel, RiskScore, PlatformComponent) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, Name = EXCLUDED.Name, Type = EXCLUDED.Type, Namespace = EXCLUDED.Namespace, NamespaceId = EXCLUDED.NamespaceId, OrchestratorComponent = EXCLUDED.OrchestratorComponent, Labels = EXCLUDED.Labels, PodLabels = EXCLUDED.PodLabels, ClusterId = EXCLUDED.ClusterId, ClusterName = EXCLUDED.ClusterName, Annotations = EXCLUDED.Annotations, Priority = EXCLUDED.Priority, ServiceAccount = EXCLUDED.ServiceAccount, ServiceAccountPermissionLevel = EXCLUDED.ServiceAccountPermissionLevel, RiskScore = EXCLUDED.RiskScore, PlatformComponent = EXCLUDED.PlatformComponent"
	_, err := db.Exec(ctx, finalStr, values...)

	return err
}
