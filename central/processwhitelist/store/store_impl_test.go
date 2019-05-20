package store

import (
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestProcessWhitelistStore(t *testing.T) {
	suite.Run(t, new(ProcessWhitelistStoreTestSuite))
}

type ProcessWhitelistStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *ProcessWhitelistStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp("process_whitelist_test.db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store, err = New(db)
	suite.NoError(err)
}

func (suite *ProcessWhitelistStoreTestSuite) TearDownSuite() {
	testutils.TearDownDB(suite.db)
}

func (suite *ProcessWhitelistStoreTestSuite) getAndCompare(whitelist *storage.ProcessWhitelist) {
	gotWhitelist, err := suite.store.GetWhitelist(whitelist.Id)
	suite.NoError(err)
	suite.Equal(whitelist, gotWhitelist)
}

func (suite *ProcessWhitelistStoreTestSuite) createAndStore() *storage.ProcessWhitelist {
	whitelist := fixtures.GetProcessWhitelistWithID()
	suite.NotNil(whitelist)
	err := suite.store.AddWhitelist(whitelist)
	suite.NoError(err)
	return whitelist
}

func (suite *ProcessWhitelistStoreTestSuite) TestGetNonExistant() {
	whitelist, err := suite.store.GetWhitelist("Not an ID")
	suite.NoError(err)
	suite.Nil(whitelist)
}

func (suite *ProcessWhitelistStoreTestSuite) TestAddGetDeleteWhitelist() {
	whitelist := suite.createAndStore()

	suite.getAndCompare(whitelist)

	err := suite.store.DeleteWhitelist(whitelist.Id)
	suite.NoError(err)
	gotWhitelist, err := suite.store.GetWhitelist(whitelist.Id)
	suite.NoError(err)
	suite.Nil(gotWhitelist)
}

func (suite *ProcessWhitelistStoreTestSuite) TestUpdateWhitelist() {
	whitelist := suite.createAndStore()

	whitelist.Elements = []*storage.WhitelistElement{fixtures.GetWhitelistElement("JosephRules")}
	err := suite.store.UpdateWhitelist(whitelist)
	suite.NoError(err)

	suite.getAndCompare(whitelist)
}

func (suite *ProcessWhitelistStoreTestSuite) TestUpdateNotExists() {
	whitelist := fixtures.GetProcessWhitelistWithID()
	err := suite.store.UpdateWhitelist(whitelist)
	suite.Error(err)
}

func (suite *ProcessWhitelistStoreTestSuite) TestAddAlreadyExists() {
	whitelist := suite.createAndStore()
	err := suite.store.AddWhitelist(whitelist)
	suite.Error(err)
}
