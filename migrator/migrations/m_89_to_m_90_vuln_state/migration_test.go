package m89tom90

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/dackboxhelpers"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/rockshelper"
	dbTypes "github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(snoozedStateMigrationTestSuite))
}

type snoozedStateMigrationTestSuite struct {
	suite.Suite

	db        *rocksdb.RocksDB
	databases *dbTypes.Databases
}

func (suite *snoozedStateMigrationTestSuite) SetupTest() {
	rocksDB, err := rocksdb.NewTemp(suite.T().Name())
	suite.NoError(err)

	suite.db = rocksDB
	suite.databases = &dbTypes.Databases{RocksDB: rocksDB.DB}
}

func (suite *snoozedStateMigrationTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *snoozedStateMigrationTestSuite) TestImagesCVEEdgeMigration() {
	cves := []*storage.CVE{
		{
			Id:         "cve1",
			Suppressed: true,
		},
		{
			Id:         "cve2",
			Suppressed: true,
		},
		{
			Id:         "cve3",
			Suppressed: false,
		},
	}

	edgeID1 := dackboxhelpers.EdgeID{ParentID: "img1", ChildID: "cve1"}.ToString()
	edgeID2 := dackboxhelpers.EdgeID{ParentID: "img1", ChildID: "cve2"}.ToString()
	edgeID3 := dackboxhelpers.EdgeID{ParentID: "img2", ChildID: "cve1"}.ToString()
	edgeID4 := dackboxhelpers.EdgeID{ParentID: "img2", ChildID: "cve3"}.ToString()
	edgeID5 := dackboxhelpers.EdgeID{ParentID: "img3", ChildID: "cve4"}.ToString()
	imageCVEEdges := []*storage.ImageCVEEdge{
		{
			Id: edgeID1,
		},
		{
			Id: edgeID2,
		},
		{
			Id: edgeID3,
		},
		{
			Id: edgeID4,
		},
		{
			Id: edgeID5,
		},
	}

	expectedUpdatedEdges := []string{edgeID1, edgeID2, edgeID3}

	for _, obj := range cves {
		key := rocksdbmigration.GetPrefixedKey(cvePrefix, []byte(obj.GetId()))
		value, err := proto.Marshal(obj)
		suite.NoError(err)
		suite.NoError(suite.databases.RocksDB.Put(writeOpts, key, value))
	}

	for _, obj := range imageCVEEdges {
		key := rocksdbmigration.GetPrefixedKey(imageCVEEdgePrefix, []byte(obj.GetId()))
		value, err := proto.Marshal(obj)
		suite.NoError(err)
		suite.NoError(suite.databases.RocksDB.Put(writeOpts, key, value))
	}

	err := updateImageCVEEdgesWithVulnState(suite.databases)
	suite.NoError(err)

	for _, id := range expectedUpdatedEdges {
		msg, exists, err := rockshelper.ReadFromRocksDB(suite.databases.RocksDB, readOpts, &storage.ImageCVEEdge{}, imageCVEEdgePrefix, []byte(id))
		suite.NoError(err)
		suite.True(exists)
		edge := msg.(*storage.ImageCVEEdge)
		suite.EqualValues(edge.GetState(), storage.VulnerabilityState_DEFERRED)
	}
}
