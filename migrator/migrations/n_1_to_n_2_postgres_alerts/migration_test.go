package n1ton2

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
	"github.com/tecbot/gorocksdb"
	"gorm.io/gorm"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(postgresMigrationSuite))
}

type postgresMigrationSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
	ctx         context.Context

	// RocksDB
	rocksDB *rocksdb.RocksDB
	db      *gorocksdb.DB

	// PostgresDB
	pool   *pgxpool.Pool
	gormDB *gorm.DB
}

var _ suite.TearDownTestSuite = (*postgresMigrationSuite)(nil)

func (s *postgresMigrationSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")
	if !features.PostgresDatastore.Enabled() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}

	var err error
	s.rocksDB, err = rocksdb.NewTemp(s.T().Name())
	s.NoError(err)

	s.db = s.rocksDB.DB

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.Require().NoError(err)

	s.ctx = context.Background()
	s.pool, err = pgxpool.ConnectConfig(s.ctx, config)
	s.Require().NoError(err)

	s.gormDB = pgtest.OpenGormDB(s.T(), source)
	_ = s.gormDB.Migrator().DropTable(pkgSchema.CreateTableAlertsStmt.GormModel)
}

func (s *postgresMigrationSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(s.rocksDB)
	_ = s.gormDB.Migrator().DropTable(pkgSchema.CreateTableAlertsStmt.GormModel)
	pgtest.CloseGormDB(s.T(), s.gormDB)
}

func (s *postgresMigrationSuite) TestMigration() {
	// Prepare data and write to legacy DB
	batchSize = 48
	rocksWriteBatch := gorocksdb.NewWriteBatch()
	defer rocksWriteBatch.Destroy()
	var objs []*storage.Alert
	for i := 0; i < 200; i++ {
		obj := &storage.Alert{}
		s.NoError(testutils.FullInit(obj, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		bytes, err := proto.Marshal(obj)
		s.NoError(err, "failed to marshal data")
		rocksWriteBatch.Put(rocksdbmigration.GetPrefixedKey(rocksdbBucket, []byte(obj.Id)), bytes)
		objs = append(objs, obj)
	}
	s.NoError(s.db.Write(gorocksdb.NewDefaultWriteOptions(), rocksWriteBatch))
	// Test migration
	s.NoError(moveAlerts(s.rocksDB, s.gormDB, s.pool))
	s.verify(objs)
	// Test re-entry
	s.NoError(moveAlerts(s.rocksDB, s.gormDB, s.pool))
	s.verify(objs)
}

func (s *postgresMigrationSuite) verify(objs []*storage.Alert) {
	var count int64
	s.gormDB.Model(pkgSchema.CreateTableAlertsStmt.GormModel).Count(&count)
	s.Equal(int64(len(objs)), count)
	for _, obj := range objs {
		s.Equal(obj, s.get(obj.Id))
	}
}

func (s *postgresMigrationSuite) get(id string) *storage.Alert {
	q := search.NewQueryBuilder().AddDocIDs(id).ProtoQuery()
	data, err := postgres.RunGetQueryForSchema(s.ctx, schema, q, s.pool)
	s.NoError(err)
	var msg storage.Alert
	s.NoError(proto.Unmarshal(data, &msg))
	return &msg
}
