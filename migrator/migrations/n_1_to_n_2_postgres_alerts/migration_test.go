package n1ton2

import (
	"context"
	"strconv"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
	"github.com/tecbot/gorocksdb"
	bolt "go.etcd.io/bbolt"
	"gorm.io/gorm"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(postgresMigrationAlertsSuite))
}

type postgresMigrationAlertsSuite struct {
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

var _ suite.TearDownTestSuite = (*postgresMigrationAlertsSuite)(nil)

func (s *postgresMigrationAlertsSuite) SetupTest() {
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
}

func (s *postgresMigrationAlertsSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(s.rocksDB)
	pgtest.OpenGormDB()
}

func (s *postgresMigrationAlertsSuite() {
	cases := []struct {
		oldSerial *storage.ServiceIdentity
		newSerial *storage.ServiceIdentity
	}{
		{
			oldSerial: &storage.ServiceIdentity{
				Srl: &storage.ServiceIdentity_Serial{
					Serial: 12345,
				},
				Id: "case1",
			},
			newSerial: &storage.ServiceIdentity{
				SerialStr: "12345",
				Srl: &storage.ServiceIdentity_Serial{
					Serial: 12345,
				},
				Id: "case1",
			},
		},
		{
			oldSerial: &storage.ServiceIdentity{
				SerialStr: "ABC",
				Id:        "case2",
			},
			newSerial: &storage.ServiceIdentity{
				SerialStr: "ABC",
				Id:        "case2",
			},
		},
	}
	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bucketName)
		s.NoError(err)
		for _, c := range cases {
			si, err := c.oldSerial.Marshal()
			s.NoError(err)

			if c.oldSerial.GetSerialStr() != "" {
				s.NoError(bucket.Put([]byte(c.oldSerial.GetSerialStr()), si))
			} else {
				s.NoError(bucket.Put([]byte(strconv.FormatInt(c.oldSerial.GetSerial(), 10)), si))
			}
		}
		return nil
	})
	s.NoError(err)

	s.NoError(migrateSerials(s.db))

	err = s.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)

		idx := 0
		err := bucket.ForEach(func(k, v []byte) error {
			var si storage.ServiceIdentity
			s.NoError(proto.Unmarshal(v, &si))
			s.Equal(cases[idx].newSerial, &si)
			idx++
			return nil
		})
		s.NoError(err)
		return nil
	})
	s.NoError(err)
}
