package m104tom105

import (
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/rockshelper"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func composeID(deploymentID, componentID string) string {
	return fmt.Sprintf("%s:%s", deploymentID, componentID)
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(activeComponentMigrationTestSuite))
}

type activeComponentMigrationTestSuite struct {
	suite.Suite

	db *rocksdb.RocksDB
}

func (suite *activeComponentMigrationTestSuite) SetupTest() {
	rocksDB, err := rocksdb.NewTemp(suite.T().Name())
	suite.NoError(err)

	suite.db = rocksDB
}

func (suite *activeComponentMigrationTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *activeComponentMigrationTestSuite) TestMigration() {
	initialActiveComponents := []*storage.ActiveComponent{
		{
			Id:                       composeID("1", "abc"),
			DEPRECATEDActiveContexts: nil,
		},
		{
			Id:                       composeID("2", "abc"),
			DEPRECATEDActiveContexts: map[string]*storage.ActiveComponent_ActiveContext{},
		},
		{
			Id: composeID("3", "abc"),
			DEPRECATEDActiveContexts: map[string]*storage.ActiveComponent_ActiveContext{
				"container1": {
					ContainerName: "container1",
					ImageId:       "image1",
				},
			},
		},
		{
			Id: composeID("4", "abc"),
			DEPRECATEDActiveContexts: map[string]*storage.ActiveComponent_ActiveContext{
				"container1": {
					ContainerName: "container1",
					ImageId:       "image1",
				},
				"container2": {
					ContainerName: "container2",
					ImageId:       "image2",
				},
			},
		},
	}
	expectedComponents := []*storage.ActiveComponent{
		{
			Id:           composeID("1", "abc"),
			DeploymentId: "1",
			ComponentId:  "abc",
		},
		{
			Id:           composeID("2", "abc"),
			DeploymentId: "2",
			ComponentId:  "abc",
		},
		{
			Id:           composeID("3", "abc"),
			DeploymentId: "3",
			ComponentId:  "abc",
			ActiveContextsSlice: []*storage.ActiveComponent_ActiveContext{
				{
					ContainerName: "container1",
					ImageId:       "image1",
				},
			},
		},
		{
			Id:           composeID("4", "abc"),
			DeploymentId: "4",
			ComponentId:  "abc",
			ActiveContextsSlice: []*storage.ActiveComponent_ActiveContext{
				{
					ContainerName: "container1",
					ImageId:       "image1",
				},
				{
					ContainerName: "container2",
					ImageId:       "image2",
				},
			},
		},
	}
	for _, initial := range initialActiveComponents {
		data, err := proto.Marshal(initial)
		suite.NoError(err)

		key := rocksdbmigration.GetPrefixedKey(activeComponentsPrefix, []byte(initial.GetId()))
		suite.NoError(suite.db.Put(writeOpts, key, data))
	}

	// Run migration
	suite.NoError(updateActiveComponents(suite.db.DB))

	for _, expected := range expectedComponents {
		msg, exists, err := rockshelper.ReadFromRocksDB(suite.db.DB, readOpts, &storage.ActiveComponent{}, activeComponentsPrefix, []byte(expected.GetId()))
		suite.NoError(err)
		suite.True(exists)

		suite.Equal(expected, msg.(*storage.ActiveComponent))
	}
}
