package m45tom46

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
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
	suite.Run(t, new(imagesCVEEdgeMigrationTestSuite))
}

type imagesCVEEdgeMigrationTestSuite struct {
	suite.Suite

	db        *rocksdb.RocksDB
	databases *dbTypes.Databases
}

func (suite *imagesCVEEdgeMigrationTestSuite) SetupTest() {
	rocksDB, err := rocksdb.NewTemp(suite.T().Name())
	suite.NoError(err)

	suite.db = rocksDB
	suite.databases = &dbTypes.Databases{RocksDB: rocksDB.DB}
}

func (suite *imagesCVEEdgeMigrationTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *imagesCVEEdgeMigrationTestSuite) TestImagesCVEEdgeMigration() {
	ts1 := types.TimestampNow()
	ts2 := types.TimestampNow()
	ts2.Seconds = ts2.Seconds + 5

	images := []*storage.Image{
		{
			Id: "sha1",
			Name: &storage.ImageName{
				FullName: "name1",
			},
			Metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Created: ts1,
				},
			},
			Scan: &storage.ImageScan{
				ScanTime: ts1,
			},
		},
		{
			Id: "sha2",
			Name: &storage.ImageName{
				FullName: "name2",
			},
			Metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Created: ts2,
				},
			},
			Scan: &storage.ImageScan{
				ScanTime: ts2,
			},
		},
		{
			Id: "sha3",
			Name: &storage.ImageName{
				FullName: "name2",
			},
			Metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Created: ts2,
				},
			},
		},
	}

	imageComponentEdges := []*storage.ImageComponentEdge{
		{
			Id: dackboxhelpers.EdgeID{
				ParentID: "sha1",
				ChildID:  "comp11",
			}.ToString(),
		},
		{
			Id: dackboxhelpers.EdgeID{
				ParentID: "sha1",
				ChildID:  "comp12",
			}.ToString(),
		},
		{
			Id: dackboxhelpers.EdgeID{
				ParentID: "sha2",
				ChildID:  "comp21",
			}.ToString(),
		},
	}

	componentCVEEdges := []*storage.ComponentCVEEdge{
		{
			Id: dackboxhelpers.EdgeID{
				ParentID: "comp11",
				ChildID:  "cve111",
			}.ToString(),
		},
		{
			Id: dackboxhelpers.EdgeID{
				ParentID: "comp21",
				ChildID:  "cve211",
			}.ToString(),
		},
		{
			Id: dackboxhelpers.EdgeID{
				ParentID: "comp21",
				ChildID:  "cve212",
			}.ToString(),
		},
	}

	cases := []struct {
		imageCVEEdge *storage.ImageCVEEdge
		exists       bool
	}{
		{
			imageCVEEdge: &storage.ImageCVEEdge{
				Id: dackboxhelpers.EdgeID{
					ParentID: "sha1",
					ChildID:  "cve111",
				}.ToString(),
				FirstImageOccurrence: ts1,
			},
			exists: true,
		},
		{
			imageCVEEdge: &storage.ImageCVEEdge{
				Id: dackboxhelpers.EdgeID{
					ParentID: "sha1",
					ChildID:  "cve111",
				}.ToString(),
				FirstImageOccurrence: ts1,
			},
			exists: true,
		},
		{
			imageCVEEdge: &storage.ImageCVEEdge{
				Id: dackboxhelpers.EdgeID{
					ParentID: "sha1",
					ChildID:  "cve111",
				}.ToString(),
				FirstImageOccurrence: ts1,
			},
			exists: true,
		},
		{
			imageCVEEdge: &storage.ImageCVEEdge{
				Id: dackboxhelpers.EdgeID{
					ParentID: "sha3",
					ChildID:  "",
				}.ToString(),
				FirstImageOccurrence: ts1,
			},
			exists: false,
		},
	}

	for _, obj := range images {
		key := rocksdbmigration.GetPrefixedKey(imagePrefix, []byte(obj.GetId()))
		value, err := proto.Marshal(obj)
		suite.NoError(err)
		suite.NoError(suite.databases.RocksDB.Put(writeOpts, key, value))
	}

	for _, obj := range imageComponentEdges {
		key := rocksdbmigration.GetPrefixedKey(imageToComponentsPrefix, []byte(obj.GetId()))
		value, err := proto.Marshal(obj)
		suite.NoError(err)
		suite.NoError(suite.databases.RocksDB.Put(writeOpts, key, value))
	}

	for _, obj := range componentCVEEdges {
		key := rocksdbmigration.GetPrefixedKey(componentsToCVEsPrefix, []byte(obj.GetId()))
		value, err := proto.Marshal(obj)
		suite.NoError(err)
		suite.NoError(suite.databases.RocksDB.Put(writeOpts, key, value))
	}

	err := writeImageCVEEdges(suite.databases)
	suite.NoError(err)

	for _, c := range cases {
		msg, exists, err := rockshelper.ReadFromRocksDB(suite.databases.RocksDB, readOpts, &storage.ImageCVEEdge{}, imageToCVEPrefix, []byte(c.imageCVEEdge.GetId()))
		suite.NoError(err)
		suite.Equal(c.exists, exists)
		if c.exists {
			suite.EqualValues(c.imageCVEEdge, msg.(*storage.ImageCVEEdge))
		}
	}
}
