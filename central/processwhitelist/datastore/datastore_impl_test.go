package datastore

import (
	"testing"

	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/processwhitelist/index"
	whitelistSearch "github.com/stackrox/rox/central/processwhitelist/search"
	"github.com/stackrox/rox/central/processwhitelist/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestProcessWhitelistDatastore(t *testing.T) {
	suite.Run(t, new(ProcessWhitelistDataStoreTestSuite))
}

type ProcessWhitelistDataStoreTestSuite struct {
	suite.Suite
	datastore DataStore
	storage   store.Store
	indexer   index.Indexer
	searcher  whitelistSearch.Searcher
}

func (suite *ProcessWhitelistDataStoreTestSuite) SetupTest() {
	db, err := bolthelper.NewTemp(testutils.DBFileName(suite.Suite))
	suite.NoError(err)
	suite.storage = store.New(db)

	tmpIndex, err := globalindex.TempInitializeIndices("")
	suite.NoError(err)
	suite.indexer = index.New(tmpIndex)

	suite.searcher, err = whitelistSearch.New(suite.storage, suite.indexer)
	suite.NoError(err)

	suite.datastore = New(suite.storage, suite.indexer, suite.searcher)
}

func (suite *ProcessWhitelistDataStoreTestSuite) mustSerializeKey(key *storage.ProcessWhitelistKey) string {
	serialized, err := keyToID(key)
	suite.Require().NoError(err)
	return string(serialized)
}

func (suite *ProcessWhitelistDataStoreTestSuite) createAndStoreWhitelist(key *storage.ProcessWhitelistKey) *storage.ProcessWhitelist {
	whitelist := fixtures.GetProcessWhitelist()
	whitelist.Key = key
	suite.NotNil(whitelist)
	id, err := suite.datastore.AddProcessWhitelist(whitelist)
	suite.NoError(err)
	suite.NotNil(id)

	suite.Equal(suite.mustSerializeKey(key), id)
	suite.Equal(id, whitelist.Id)
	return whitelist
}

func (suite *ProcessWhitelistDataStoreTestSuite) createAndStoreWhitelistWithRandomKey() *storage.ProcessWhitelist {
	return suite.createAndStoreWhitelist(&storage.ProcessWhitelistKey{
		DeploymentId:  uuid.NewV4().String(),
		ContainerName: uuid.NewV4().String(),
	})
}

func (suite *ProcessWhitelistDataStoreTestSuite) testGetAll(allExpected []*storage.ProcessWhitelist) {
	allWhitelists, err := suite.datastore.GetProcessWhitelists()
	suite.NoError(err)
	suite.NotNil(allWhitelists)
	suite.Equal(len(allExpected), len(allWhitelists))
	for _, expected := range allExpected {
		suite.Contains(allWhitelists, expected)
	}
}

func (suite *ProcessWhitelistDataStoreTestSuite) TestGetById() {
	gotWhitelist, err := suite.datastore.GetProcessWhitelist(&storage.ProcessWhitelistKey{DeploymentId: "FAKE", ContainerName: "whatever"})
	suite.NoError(err)
	suite.Nil(gotWhitelist)

	key := &storage.ProcessWhitelistKey{
		DeploymentId:  "blah",
		ContainerName: "container",
	}
	whitelist := suite.createAndStoreWhitelist(key)
	gotWhitelist, err = suite.datastore.GetProcessWhitelist(key)
	suite.NoError(err)
	suite.Equal(whitelist, gotWhitelist)
}

func (suite *ProcessWhitelistDataStoreTestSuite) TestGetAllWhitelists() {
	expectedWhitelists := []*storage.ProcessWhitelist{suite.createAndStoreWhitelistWithRandomKey(), suite.createAndStoreWhitelistWithRandomKey()}
	suite.testGetAll(expectedWhitelists)
}

func (suite *ProcessWhitelistDataStoreTestSuite) TestRemoveProcessWhitelist() {
	whitelist := suite.createAndStoreWhitelistWithRandomKey()
	key := whitelist.GetKey()
	gotWhitelist, err := suite.datastore.GetProcessWhitelist(whitelist.GetKey())
	suite.NoError(err)
	suite.Equal(whitelist, gotWhitelist)
	err = suite.datastore.RemoveProcessWhitelist(key)
	suite.NoError(err)
	gotWhitelist, err = suite.datastore.GetProcessWhitelist(key)
	suite.NoError(err)
	suite.Nil(gotWhitelist)
}

func (suite *ProcessWhitelistDataStoreTestSuite) TestLockAndUnlockWhitelist() {
	whitelist := suite.createAndStoreWhitelistWithRandomKey()
	key := whitelist.GetKey()
	suite.Nil(whitelist.GetUserLockedTimestamp())
	updatedWhitelist, err := suite.datastore.UserLockProcessWhitelist(key, true)
	suite.NoError(err)
	suite.NotNil(updatedWhitelist.GetUserLockedTimestamp())
	gotWhitelist, err := suite.datastore.GetProcessWhitelist(key)
	suite.NoError(err)
	suite.Equal(updatedWhitelist, gotWhitelist)

	updatedWhitelist, err = suite.datastore.UserLockProcessWhitelist(key, false)
	suite.NoError(err)
	suite.Nil(updatedWhitelist.GetUserLockedTimestamp())
	gotWhitelist, err = suite.datastore.GetProcessWhitelist(key)
	suite.NoError(err)
	suite.Equal(whitelist, gotWhitelist)
}

func (suite *ProcessWhitelistDataStoreTestSuite) TestRoxLockAndUnlockWhitelist() {
	whitelist := suite.createAndStoreWhitelistWithRandomKey()
	key := whitelist.GetKey()
	suite.Nil(whitelist.GetStackRoxLockedTimestamp())
	updatedWhitelist, err := suite.datastore.RoxLockProcessWhitelist(key, true)
	suite.NoError(err)
	suite.NotNil(updatedWhitelist.GetStackRoxLockedTimestamp())
	gotWhitelist, err := suite.datastore.GetProcessWhitelist(key)
	suite.NoError(err)
	suite.Equal(updatedWhitelist, gotWhitelist)

	updatedWhitelist, err = suite.datastore.RoxLockProcessWhitelist(key, false)
	suite.NoError(err)
	suite.Nil(updatedWhitelist.GetStackRoxLockedTimestamp())
	gotWhitelist, err = suite.datastore.GetProcessWhitelist(key)
	suite.NoError(err)
	suite.Equal(whitelist, gotWhitelist)
}

func (suite *ProcessWhitelistDataStoreTestSuite) TestChangeAutoToManual() {
	whitelist := fixtures.GetProcessWhitelist()
	suite.NotNil(whitelist)
	suite.NotNil(whitelist.Elements)
	suite.Equal(1, len(whitelist.Elements))
	processName := whitelist.Elements[0].GetProcessName()
	whitelist.Elements[0].Auto = true
	whitelist.Key = &storage.ProcessWhitelistKey{DeploymentId: "blah", ContainerName: "blah2"}
	id, err := suite.datastore.AddProcessWhitelist(whitelist)
	suite.NoError(err)
	suite.NotNil(id)

	updated, err := suite.datastore.UpdateProcessWhitelist(whitelist.GetKey(), []string{processName}, nil)
	suite.NoError(err)
	suite.NotNil(updated)
	suite.NotNil(updated.Elements)
	suite.Equal(1, len(updated.Elements))
	suite.Equal(processName, updated.Elements[0].GetProcessName())
	suite.False(updated.Elements[0].Auto)
}
