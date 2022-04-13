package m91tom92

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/migrator/migrations/dackboxhelpers"
	"github.com/stackrox/stackrox/migrator/migrations/m_91_to_m_92_write_edges_to_graph/sortedkeys"
	"github.com/stackrox/stackrox/migrator/migrations/rocksdbmigration"
	dbTypes "github.com/stackrox/stackrox/migrator/types"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
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
	img1 := "img1"
	img2 := "img2"
	img3 := "img3"
	edges := map[string]map[string]struct{}{
		string(img1): {
			"cve1": {},
			"cve2": {},
		},
		string(img2): {
			"cve1": {},
			"cve3": {},
			"cve4": {},
		},
		string(img3): {
			"cve4": {},
		},
	}

	for image, cves := range edges {
		for cve := range cves {
			edge := &storage.ImageCVEEdge{Id: dackboxhelpers.EdgeID{ParentID: image, ChildID: cve}.ToString()}
			key := rocksdbmigration.GetPrefixedKey(imageCVEEdgePrefix, []byte(edge.GetId()))
			value, err := proto.Marshal(edge)
			suite.NoError(err)
			suite.NoError(suite.databases.RocksDB.Put(writeOpts, key, value))
		}
	}

	err := writeImageCVEEdgesToGraph(suite.databases)
	suite.NoError(err)

	it := suite.databases.RocksDB.NewIterator(readOpts)
	defer it.Close()

	count := 0
	for it.Seek(graphBucket); it.ValidForPrefix(graphBucket); it.Next() {
		key := it.Key().Copy()
		id := rocksdbmigration.GetIDFromPrefixedKey(graphBucket, key)
		tos, err := sortedkeys.Unmarshal(it.Value().Data())
		suite.NoError(err)

		imageKey := rocksdbmigration.GetIDFromPrefixedKey(imageBucket, id)
		expectedTos, ok := edges[string(imageKey)]
		suite.True(ok)
		for to := range expectedTos {
			suite.NotEqual(-1, tos.Find(getCVEKey([]byte(to))))
		}
		count++
	}
	suite.Equal(3, count)
}
