package datastore

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/processwhitelist/index"
	whitelistSearch "github.com/stackrox/rox/central/processwhitelist/search"
	"github.com/stackrox/rox/central/processwhitelist/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/testutils"
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

func (suite *ProcessWhitelistDataStoreTestSuite) createAndStoreWhitelist() *storage.ProcessWhitelist {
	whitelist := fixtures.GetProcessWhitelist()
	suite.NotNil(whitelist)
	id, err := suite.datastore.AddProcessWhitelist(whitelist)
	suite.NoError(err)
	suite.NotNil(id)
	suite.Equal(fmt.Sprintf("%s/%s", whitelist.DeploymentId, whitelist.ContainerName), id)
	suite.Equal(id, whitelist.Id)
	return whitelist
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
	gotWhitelist, err := suite.datastore.GetProcessWhitelist("Not an ID")
	suite.NoError(err)
	suite.Nil(gotWhitelist)

	whitelist := suite.createAndStoreWhitelist()
	gotWhitelist, err = suite.datastore.GetProcessWhitelist(whitelist.Id)
	suite.NoError(err)
	suite.Equal(whitelist, gotWhitelist)
}

func (suite *ProcessWhitelistDataStoreTestSuite) TestGetAllWhitelists() {
	expectedWhitelists := []*storage.ProcessWhitelist{suite.createAndStoreWhitelist(), suite.createAndStoreWhitelist()}
	suite.testGetAll(expectedWhitelists)
}

func (suite *ProcessWhitelistDataStoreTestSuite) TestRemoveProcessWhitelist() {
	whitelist := suite.createAndStoreWhitelist()
	gotWhitelist, err := suite.datastore.GetProcessWhitelist(whitelist.Id)
	suite.NoError(err)
	suite.Equal(whitelist, gotWhitelist)
	err = suite.datastore.RemoveProcessWhitelist(whitelist.Id)
	suite.NoError(err)
	gotWhitelist, err = suite.datastore.GetProcessWhitelist(whitelist.Id)
	suite.NoError(err)
	suite.Nil(gotWhitelist)
}

func (suite *ProcessWhitelistDataStoreTestSuite) TestLockAndUnlockWhitelist() {
	whitelist := suite.createAndStoreWhitelist()
	suite.Nil(whitelist.GetUserLockedTimestamp())
	updatedWhitelist, err := suite.datastore.UserLockProcessWhitelist(whitelist.GetId(), true)
	suite.NoError(err)
	suite.NotNil(updatedWhitelist.GetUserLockedTimestamp())
	gotWhitelist, err := suite.datastore.GetProcessWhitelist(whitelist.GetId())
	suite.NoError(err)
	suite.Equal(updatedWhitelist, gotWhitelist)

	updatedWhitelist, err = suite.datastore.UserLockProcessWhitelist(whitelist.GetId(), false)
	suite.NoError(err)
	suite.Nil(updatedWhitelist.GetUserLockedTimestamp())
	gotWhitelist, err = suite.datastore.GetProcessWhitelist(whitelist.GetId())
	suite.NoError(err)
	suite.Equal(whitelist, gotWhitelist)
}

func (suite *ProcessWhitelistDataStoreTestSuite) TestRoxLockAndUnlockWhitelist() {
	whitelist := suite.createAndStoreWhitelist()
	suite.Nil(whitelist.GetStackRoxLockedTimestamp())
	updatedWhitelist, err := suite.datastore.RoxLockProcessWhitelist(whitelist.GetId(), true)
	suite.NoError(err)
	suite.NotNil(updatedWhitelist.GetStackRoxLockedTimestamp())
	gotWhitelist, err := suite.datastore.GetProcessWhitelist(whitelist.GetId())
	suite.NoError(err)
	suite.Equal(updatedWhitelist, gotWhitelist)

	updatedWhitelist, err = suite.datastore.RoxLockProcessWhitelist(whitelist.GetId(), false)
	suite.NoError(err)
	suite.Nil(updatedWhitelist.GetStackRoxLockedTimestamp())
	gotWhitelist, err = suite.datastore.GetProcessWhitelist(whitelist.GetId())
	suite.NoError(err)
	suite.Equal(whitelist, gotWhitelist)
}

func (suite *ProcessWhitelistDataStoreTestSuite) TestChangeAutoToManual() {
	whitelist := fixtures.GetProcessWhitelist()
	suite.NotNil(whitelist)
	suite.NotNil(whitelist.Processes)
	suite.Equal(1, len(whitelist.Processes))
	processName := whitelist.Processes[0].Name
	whitelist.Processes[0].Auto = true
	id, err := suite.datastore.AddProcessWhitelist(whitelist)
	suite.NoError(err)
	suite.NotNil(id)

	updated, err := suite.datastore.UpdateProcessWhitelist(id, []string{processName}, nil)
	suite.NoError(err)
	suite.NotNil(updated)
	suite.NotNil(updated.Processes)
	suite.Equal(1, len(updated.Processes))
	suite.Equal(processName, updated.Processes[0].Name)
	suite.False(updated.Processes[0].Auto)
}
