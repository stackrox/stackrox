//go:build sql_integration

package n23ton24

import (
	"context"
	"strconv"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	legacy "github.com/stackrox/rox/migrator/migrations/n_23_to_n_24_postgres_image_integrations/legacy"
	pgStore "github.com/stackrox/rox/migrator/migrations/n_23_to_n_24_postgres_image_integrations/postgres"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(postgresMigrationSuite))
}

type postgresMigrationSuite struct {
	suite.Suite
	ctx context.Context

	legacyDB   *bolt.DB
	postgresDB *pghelper.TestPostgres
}

var _ suite.TearDownTestSuite = (*postgresMigrationSuite)(nil)

func (s *postgresMigrationSuite) SetupTest() {
	var err error
	s.legacyDB, err = bolthelper.NewTemp(s.T().Name() + ".db")
	s.NoError(err)

	s.Require().NoError(err)

	s.ctx = sac.WithAllAccess(context.Background())
	s.postgresDB = pghelper.ForT(s.T(), false)
}

func (s *postgresMigrationSuite) TearDownTest() {
	testutils.TearDownDB(s.legacyDB)
	s.postgresDB.Teardown(s.T())
}

func (s *postgresMigrationSuite) TestImageIntegrationMigration() {
	newStore := pgStore.New(s.postgresDB.DB)
	legacyStore := legacy.New(s.legacyDB)

	// Prepare data and write to legacy DB
	var imageIntegrations []*storage.ImageIntegration
	countBadIDs := 0
	for i := 0; i < 200; i++ {
		imageIntegration := &storage.ImageIntegration{}
		s.NoError(testutils.FullInit(imageIntegration, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		imageIntegrations = append(imageIntegrations, imageIntegration)
		if i%10 == 0 {
			imageIntegration.Id = strconv.Itoa(i)
			countBadIDs = countBadIDs + 1
		}
		s.NoError(legacyStore.Upsert(s.ctx, imageIntegration))
	}

	// Move
	s.NoError(move(s.ctx, s.postgresDB.GetGormDB(), s.postgresDB.DB, legacyStore))

	// Verify
	count, err := newStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(len(imageIntegrations)-countBadIDs, count)
	for _, imageIntegration := range imageIntegrations {
		if pgutils.NilOrUUID(imageIntegration.GetId()) != nil {
			fetched, exists, err := newStore.Get(s.ctx, imageIntegration.GetId())
			s.NoError(err)
			s.True(exists)
			s.Equal(imageIntegration, fetched)
		}
	}
}
