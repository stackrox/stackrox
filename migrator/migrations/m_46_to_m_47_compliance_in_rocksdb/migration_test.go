package m45tom46

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	dbTypes "github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/rocksdb"
	generic "github.com/stackrox/rox/pkg/rocksdb/crud"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(clusterRocksDBMigrationTestSuite))
}

type clusterRocksDBMigrationTestSuite struct {
	suite.Suite

	db        *rocksdb.RocksDB
	databases *dbTypes.Databases
}

func (s *clusterRocksDBMigrationTestSuite) SetupTest() {
	rocksDB, err := rocksdb.NewTemp(s.T().Name())
	s.NoError(err)

	s.db = rocksDB
	s.databases = &dbTypes.Databases{RocksDB: rocksDB.DB}
}

func (s *clusterRocksDBMigrationTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(s.db)
}

func getMockComplianceRunResults() (*storage.ComplianceRunMetadata, *storage.ComplianceRunResults, *storage.ComplianceStrings) {
	res := &storage.ComplianceRunResults{
		Domain: &storage.ComplianceDomain{
			Cluster: &storage.Cluster{
				Id:   "testcluster",
				Name: "Joseph Rules",
			},
			Nodes: map[string]*storage.Node{
				"node": {
					Id:   "testnode",
					Name: "Joseph is the best",
				},
			},
			Deployments: map[string]*storage.Deployment{
				"deployment": {
					Id:   "testdeployment",
					Name: "Joseph is super cool",
				},
			},
		},
		RunMetadata: &storage.ComplianceRunMetadata{
			RunId:           "abcd",
			StandardId:      "efgh",
			ClusterId:       "jklm",
			StartTimestamp:  types.TimestampNow(),
			FinishTimestamp: types.TimestampNow(),
			Success:         true,
		},
		ClusterResults: &storage.ComplianceRunResults_EntityResults{
			ControlResults: map[string]*storage.ComplianceResultValue{
				"some result": {
					Evidence: []*storage.ComplianceResultValue_Evidence{
						{
							MessageId: 1,
						},
					},
				},
			},
		},
	}

	complianceStrings := &storage.ComplianceStrings{
		Strings: []string{
			"wooooo",
		},
	}

	return res.RunMetadata, res, complianceStrings
}

func (s *clusterRocksDBMigrationTestSuite) TestComplianceRocksDBMigration() {
	md, res, str := getMockComplianceRunResults()
	mdKey, resKey, strKey, err := makeKeys(md.ClusterId, md.StandardId, md.RunId, md.FinishTimestamp)
	s.Require().NoError(err)
	err = storeRun(md, res, str, s.databases.RocksDB)
	s.Require().NoError(err)

	mdSlice, err := s.databases.RocksDB.Get(generic.DefaultReadOptions(), mdKey)
	s.Require().NoError(err)
	s.Require().True(mdSlice.Exists())
	var dbMd storage.ComplianceRunMetadata
	err = proto.Unmarshal(mdSlice.Data(), &dbMd)
	s.Require().NoError(err)
	s.Equal(md, &dbMd)

	resSlice, err := s.databases.RocksDB.Get(generic.DefaultReadOptions(), resKey)
	s.Require().NoError(err)
	s.Require().True(resSlice.Exists())
	var dbRes storage.ComplianceRunResults
	err = proto.Unmarshal(resSlice.Data(), &dbRes)
	s.Require().NoError(err)
	s.Equal(res, &dbRes)

	strSlice, err := s.databases.RocksDB.Get(generic.DefaultReadOptions(), strKey)
	s.Require().NoError(err)
	s.Require().True(strSlice.Exists())
	var dbStr storage.ComplianceStrings
	err = proto.Unmarshal(strSlice.Data(), &dbStr)
	s.Require().NoError(err)
	s.Equal(str, &dbStr)
}
