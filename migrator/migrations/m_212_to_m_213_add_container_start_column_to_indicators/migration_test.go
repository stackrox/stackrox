//go:build sql_integration

package m212tom213

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	updatedSchema "github.com/stackrox/rox/migrator/migrations/m_212_to_m_213_add_container_start_column_to_indicators/schema"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_212_to_m_213_add_container_start_column_to_indicators/test/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
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
	// s.db = pghelper.ForTExistingDB(s.T(), false, "indicators")
	// s.existingDB = true
}

func (s *migrationTestSuite) TestMigration() {
	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	if !s.existingDB {
		// Create the old schema for testing
		pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, oldSchema.CreateTableProcessIndicatorsStmt)
		cluster4 := uuid.NewV4().String()
		cluster5 := uuid.NewV4().String()
		cluster6 := uuid.NewV4().String()
		cluster7 := uuid.NewV4().String()
		cluster8 := uuid.NewV4().String()
		cluster9 := uuid.NewV4().String()
		cluster10 := uuid.NewV4().String()

		clusters := []string{fixtureconsts.Cluster1, fixtureconsts.Cluster2, fixtureconsts.Cluster3, cluster4, cluster5, cluster6, cluster7, cluster8, cluster9, cluster10}

		// Add some process indicators
		numIndicators := 300
		numNilContainerTime := 10
		var indicators []*storage.ProcessIndicator

		log.Info("Building base indicators")
		for i := 0; i < numIndicators; i++ {
			processIndicator := &storage.ProcessIndicator{}
			s.NoError(testutils.FullInit(processIndicator, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
			indicators = append(indicators, processIndicator)
		}
		log.Info("Built indicators")

		for _, cluster := range clusters {
			var convertedProcessIndicators []oldSchema.ProcessIndicators
			//log.Info("Writing cluster")
			for i, processIndicator := range indicators {
				// spreading these across some deployments to set up search test
				processIndicator.ClusterId = cluster
				processIndicator.Id = uuid.NewV4().String()

				// Since we are skipping records that have a nil time we need to create some to ensure that code executes properly
				if i < numNilContainerTime {
					processIndicator.ContainerStartTime = nil
				}

				converted, err := oldSchema.ConvertProcessIndicatorFromProto(processIndicator)
				s.Require().NoError(err)
				convertedProcessIndicators = append(convertedProcessIndicators, *converted)

				if len(convertedProcessIndicators) == 1000 {
					// Upsert converted blobs
					s.Require().NoError(dbs.GormDB.CreateInBatches(convertedProcessIndicators, batchSize).Error)
					convertedProcessIndicators = convertedProcessIndicators[:0]
				}
			}
			if len(convertedProcessIndicators) > 0 {
				s.Require().NoError(dbs.GormDB.CreateInBatches(convertedProcessIndicators, batchSize).Error)
			}
			//log.Info("Wrote cluster")
		}

		log.Info("Created the indicators")
	}

	// Apply the new schema to then ensure time field is empty
	pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, updatedSchema.CreateTableProcessIndicatorsStmt)

	var n int
	err := s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM process_indicators WHERE containerstarttime IS NULL;").Scan(&n)
	s.NoError(err)
	log.Infof("Found %d indicators", n)
	//s.Require().Equal(numIndicators*len(clusters), n)

	// Now run the migration
	log.Info("Start migration")
	s.Require().NoError(migration.Run(dbs))
	log.Info("End migration")

	// After the migration, timestamp should only be NULL for indicators that had a null container time in the serialized object.
	err = s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM process_indicators WHERE containerstarttime IS NULL;").Scan(&n)
	s.NoError(err)
	log.Infof("Found %d indicators with nil time", n)
	//s.Require().Equal(numNilContainerTime*len(clusters), n)
}
