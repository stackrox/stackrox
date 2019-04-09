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

func (suite *ProcessWhitelistDataStoreTestSuite) TestGetByNames() {
	gotWhitelist, err := suite.datastore.GetProcessWhitelistByNames("Not an", "ID")
	suite.NoError(err)
	suite.Nil(gotWhitelist)

	whitelist := suite.createAndStoreWhitelist()
	gotWhitelist, err = suite.datastore.GetProcessWhitelistByNames(whitelist.DeploymentId, whitelist.ContainerName)
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
