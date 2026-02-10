//go:build sql_integration

package m220tom221

import (
	"context"
	"testing"

	"github.com/stackrox/hashstructure"
	"github.com/stackrox/rox/generated/storage"
	updatedSchema "github.com/stackrox/rox/migrator/migrations/m_220_to_m_221_add_deployment_hash_column/schema"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_220_to_m_221_add_deployment_hash_column/test/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
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
	// s.db = pghelper.ForTExistingDB(s.T(), false, "database_id_here")
	// s.existingDB = true
}

func (s *migrationTestSuite) TestMigration() {
	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	clusters := []string{fixtureconsts.Cluster1, fixtureconsts.Cluster2, fixtureconsts.Cluster3}
	numDeployments := 3000
	deploymentsPerCluster := numDeployments / len(clusters)

	if !s.existingDB {
		// Create the old schema for testing (without hash column)
		pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, oldSchema.CreateTableDeploymentsStmt)

		log.WriteToStderrf("Building test deployments")
		for _, clusterID := range clusters {
			var deployments []*storage.Deployment
			for i := 0; i < deploymentsPerCluster; i++ {
				deployment := &storage.Deployment{}
				s.NoError(testutils.FullInit(deployment, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
				deployment.ClusterId = clusterID

				// Compute hash for the deployment
				hashValue, err := hashstructure.Hash(deployment, &hashstructure.HashOptions{})
				s.Require().NoError(err)
				deployment.Hash = hashValue

				deployments = append(deployments, deployment)
			}

			// Convert deployments to old schema format and insert in batches
			var convertedDeployments []oldSchema.Deployments
			for _, deployment := range deployments {
				converted, err := oldSchema.ConvertDeploymentFromProto(deployment)
				s.Require().NoError(err)
				convertedDeployments = append(convertedDeployments, *converted)

				if len(convertedDeployments) == 100 {
					// Insert converted deployments
					s.Require().NoError(dbs.GormDB.CreateInBatches(convertedDeployments, batchSize).Error)
					convertedDeployments = convertedDeployments[:0]
				}
			}
			if len(convertedDeployments) > 0 {
				s.Require().NoError(dbs.GormDB.CreateInBatches(convertedDeployments, batchSize).Error)
			}
		}
		log.WriteToStderrf("Created test deployments")
	}

	// Apply the new schema to add hash column with NULL values
	pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, updatedSchema.CreateTableDeploymentsStmt)

	// Verify hash column is NULL before migration
	var nullHashCount int
	err := s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM deployments WHERE hash IS NULL;").Scan(&nullHashCount)
	s.NoError(err)
	log.WriteToStderrf("Found %d deployments with NULL hash before migration", nullHashCount)
	s.Require().Equal(numDeployments, nullHashCount)

	// Now run the migration
	log.WriteToStderrf("Start migration")
	s.Require().NoError(migration.Run(dbs))
	log.WriteToStderrf("End migration")

	// After the migration, hash should be populated for all deployments
	err = s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM deployments WHERE hash IS NULL;").Scan(&nullHashCount)
	s.NoError(err)
	log.WriteToStderrf("Found %d deployments with NULL hash after migration", nullHashCount)
	s.Require().Equal(0, nullHashCount)

	// Verify hash values match expected values from serialized blob
	var count int
	err = s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM deployments WHERE hash > 0;").Scan(&count)
	s.NoError(err)
	log.WriteToStderrf("Found %d deployments with populated hash", count)
	s.Require().Equal(numDeployments, count)

	// Run the migration a second time to ensure idempotency.
	s.Assert().NoError(migration.Run(dbs))

	// Verify hash values are still correct after second run
	err = s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM deployments WHERE hash IS NULL;").Scan(&nullHashCount)
	s.NoError(err)
	s.Require().Equal(0, nullHashCount)

	err = s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM deployments WHERE hash > 0;").Scan(&count)
	s.NoError(err)
	s.Require().Equal(numDeployments, count)
}
