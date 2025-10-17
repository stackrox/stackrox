//go:build sql_integration

package m214tom215

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_214_to_m_215_populate_alert_deployment_containers_imageidv2/schema"
	"github.com/stackrox/rox/migrator/migrations/m_214_to_m_215_populate_alert_deployment_containers_imageidv2/store"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

type migrationTestSuite struct {
	suite.Suite

	store store.Store
	db    *pghelper.TestPostgres
	ctx   context.Context
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pghelper.ForT(s.T(), false)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTableAlertsStmt)
	s.store = store.New(s.db)
}

func (s *migrationTestSuite) TestMigration() {
	alerts := map[string]*storage.Alert{
		fixtureconsts.Alert1: {
			Id: fixtureconsts.Alert1,
			Policy: &storage.Policy{
				Id:          "policy-1",
				Name:        "Test Policy 1",
				Description: "Test policy description 1",
				Disabled:    false,
				Categories:  []string{"Test"},
				Severity:    storage.Severity_HIGH_SEVERITY,
				EnforcementActions: []storage.EnforcementAction{
					storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
				},
			},
			LifecycleStage: storage.LifecycleStage_DEPLOY,
			ClusterId:      "cluster-1",
			ClusterName:    "Test Cluster",
			Namespace:      "default",
			NamespaceId:    "ns-uuid-1",
			Entity: &storage.Alert_Deployment_{
				Deployment: &storage.Alert_Deployment{
					Id:       "deployment-1",
					Name:     "Test Deployment 1",
					Inactive: false,
					Containers: []*storage.Alert_Deployment_Container{
						{
							Name: "container-1",
							Image: &storage.ContainerImage{
								Id: "sha256:123456789abcdef123456789abcdef123456789abcdef123456789abcdef1234",
								Name: &storage.ImageName{
									Registry: "docker.io",
									Remote:   "myorg/myimage",
									Tag:      "latest",
									FullName: "docker.io/myorg/myimage:latest",
								},
								IdV2: "",
							},
						},
					},
				},
			},
		},
		fixtureconsts.Alert2: {
			Id: fixtureconsts.Alert2,
			Policy: &storage.Policy{
				Id:          "policy-2",
				Name:        "Test Policy 2",
				Description: "Test policy description 2",
				Disabled:    false,
				Categories:  []string{"Another"},
				Severity:    storage.Severity_MEDIUM_SEVERITY,
			},
			LifecycleStage: storage.LifecycleStage_DEPLOY,
			ClusterId:      "cluster-2",
			ClusterName:    "Another Cluster",
			Namespace:      "dev",
			NamespaceId:    "ns-uuid-2",
			Entity: &storage.Alert_Deployment_{
				Deployment: &storage.Alert_Deployment{
					Id:       "deployment-2",
					Name:     "Another Deployment",
					Inactive: true,
					Containers: []*storage.Alert_Deployment_Container{
						{
							Name: "container-2",
							Image: &storage.ContainerImage{
								Id: "sha256:deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
								Name: &storage.ImageName{
									Registry: "gcr.io",
									Remote:   "org/app",
									Tag:      "v1.0.0",
									FullName: "gcr.io/org/app:v1.0.0",
								},
								IdV2: "",
							},
						},
						{
							Name: "container-3",
							Image: &storage.ContainerImage{
								Id: "sha256:cafebabecafebabecafebabecafebabecafebabecafebabecafebabecafebabe",
								Name: &storage.ImageName{
									Registry: "quay.io",
									Remote:   "repo/image",
									Tag:      "stable",
									FullName: "quay.io/repo/image:stable",
								},
								IdV2: "",
							},
						},
					},
				},
			},
		},
	}

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	for _, alert := range alerts {
		err := s.store.Upsert(dbs.DBCtx, alert)
		s.Require().NoError(err)
	}

	alerts[fixtureconsts.Alert1].GetDeployment().GetContainers()[0].GetImage().IdV2 = "8abd44b6-754e-5f67-a68e-457d304b7fd5"
	alerts[fixtureconsts.Alert2].GetDeployment().GetContainers()[0].GetImage().IdV2 = "9c824579-76de-5836-9b89-51c5986384ac"
	alerts[fixtureconsts.Alert2].GetDeployment().GetContainers()[1].GetImage().IdV2 = "f6f12b0b-be0c-543e-baae-c97c3cb79584"

	s.Require().NoError(migration.Run(dbs))

	_ = s.store.WalkByQuery(dbs.DBCtx, search.EmptyQuery(), func(alert *storage.Alert) error {
		expectedAlert, found := alerts[alert.Id]
		s.Require().True(found)
		protoassert.Equal(s.T(), expectedAlert, alert)
		return nil
	})

}
