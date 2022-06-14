package m61tom62

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/rockshelper"
	dbTypes "github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/rocksdb"
	rocksdbopts "github.com/stackrox/rox/pkg/rocksdb/crud"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(multipleCVETypesMigrationTestSuite))
}

type multipleCVETypesMigrationTestSuite struct {
	suite.Suite

	db        *rocksdb.RocksDB
	databases *dbTypes.Databases
}

func (suite *multipleCVETypesMigrationTestSuite) SetupTest() {
	rocksDB := rocksdbtest.RocksDBForT(suite.T())

	suite.db = rocksDB
	suite.databases = &dbTypes.Databases{RocksDB: rocksDB.DB}
}

func (suite *multipleCVETypesMigrationTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *multipleCVETypesMigrationTestSuite) TestMultipleCVETypesMigration() {
	cves := []*storage.CVE{
		{
			Id:   "CVE-2021-1234",
			Type: storage.CVE_IMAGE_CVE,
		},
		{
			Id:   "CVE-2021-1235",
			Type: storage.CVE_NODE_CVE,
		},
		{
			Id:   "CVE-2021-1236",
			Type: storage.CVE_ISTIO_CVE,
		},
		{
			Id:   "CVE-2021-1237",
			Type: storage.CVE_K8S_CVE,
		},
		{
			Id:   "CVE-2021-1238",
			Type: storage.CVE_UNKNOWN_CVE,
		},
	}

	cases := []struct {
		cve *storage.CVE
	}{
		{
			cve: &storage.CVE{
				Id:    "CVE-2021-1234",
				Types: []storage.CVE_CVEType{storage.CVE_IMAGE_CVE},
			},
		},
		{
			cve: &storage.CVE{
				Id:    "CVE-2021-1235",
				Types: []storage.CVE_CVEType{storage.CVE_NODE_CVE},
			},
		},
		{
			cve: &storage.CVE{
				Id:    "CVE-2021-1236",
				Types: []storage.CVE_CVEType{storage.CVE_ISTIO_CVE},
			},
		},
		{
			cve: &storage.CVE{
				Id:    "CVE-2021-1237",
				Types: []storage.CVE_CVEType{storage.CVE_K8S_CVE},
			},
		},
		{
			cve: &storage.CVE{
				Id:    "CVE-2021-1238",
				Types: []storage.CVE_CVEType{storage.CVE_UNKNOWN_CVE},
			},
		},
	}

	for _, cve := range cves {
		key := rocksdbmigration.GetPrefixedKey(cveBucket, []byte(cve.GetId()))
		value, err := proto.Marshal(cve)
		suite.NoError(err)
		suite.NoError(suite.db.Put(writeOpts, key, value))
	}

	err := migrateCVEs(suite.databases.RocksDB)
	suite.NoError(err)

	for _, c := range cases {
		msg, exists, err := rockshelper.ReadFromRocksDB(suite.databases.RocksDB, rocksdbopts.DefaultReadOptions(), &storage.CVE{}, cveBucket, []byte(c.cve.GetId()))
		suite.NoError(err)
		suite.True(exists)
		suite.EqualValues(c.cve, msg.(*storage.CVE))
	}
}
