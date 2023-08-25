//go:build sql_integration

package n2ton3

import (
	"context"
	"strconv"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	legacy "github.com/stackrox/rox/migrator/migrations/n_02_to_n_03_postgres_namespaces/legacy"
	pgStore "github.com/stackrox/rox/migrator/migrations/n_02_to_n_03_postgres_namespaces/postgres"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(postgresMigrationSuite))
}

type postgresMigrationSuite struct {
	suite.Suite
	ctx context.Context

	legacyDB   *rocksdb.RocksDB
	postgresDB *pghelper.TestPostgres
}

var _ suite.TearDownTestSuite = (*postgresMigrationSuite)(nil)

func (s *postgresMigrationSuite) SetupTest() {
	var err error
	s.legacyDB, err = rocksdb.NewTemp(s.T().Name())
	s.NoError(err)

	s.Require().NoError(err)

	s.ctx = sac.WithAllAccess(context.Background())
	s.postgresDB = pghelper.ForT(s.T(), false)
}

func (s *postgresMigrationSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(s.legacyDB)
	s.postgresDB.Teardown(s.T())
}

func (s *postgresMigrationSuite) TestNamespaceMetadataMigration() {
	newStore := pgStore.New(s.postgresDB.DB)
	legacyStore, err := legacy.New(s.legacyDB)
	s.NoError(err)

	// Prepare data and write to legacy DB
	var namespaceMetadatas []*storage.NamespaceMetadata
	countBadIDs := 0
	for i := 0; i < 200; i++ {
		namespaceMetadata := &storage.NamespaceMetadata{}
		s.NoError(testutils.FullInit(namespaceMetadata, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		if i%10 == 0 {
			namespaceMetadata.Id = strconv.Itoa(i)
			countBadIDs = countBadIDs + 1
		}
		namespaceMetadatas = append(namespaceMetadatas, namespaceMetadata)
	}

	s.NoError(legacyStore.UpsertMany(s.ctx, namespaceMetadatas))

	// Move
	s.NoError(move(s.ctx, s.postgresDB.GetGormDB(), s.postgresDB.DB, legacyStore))

	// Verify
	count, err := newStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(len(namespaceMetadatas)-countBadIDs, count)
	for _, namespaceMetadata := range namespaceMetadatas {
		if pgutils.NilOrUUID(namespaceMetadata.GetId()) != nil {
			fetched, exists, err := newStore.Get(s.ctx, namespaceMetadata.GetId())
			s.NoError(err)
			s.True(exists)
			s.Equal(namespaceMetadata, fetched)
		}
	}
}
