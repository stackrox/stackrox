package bolt

import (
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/central/processwhitelist/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/storecache"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestProcessWhitelistStore(t *testing.T) {
	suite.Run(t, new(ProcessWhitelistStoreTestSuite))
}

type ProcessWhitelistStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store store.Store
}

func (suite *ProcessWhitelistStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp("process_whitelist_test.db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store, err = NewStore(db, storecache.NewMapBackedCache())
	suite.NoError(err)
}

func (suite *ProcessWhitelistStoreTestSuite) TearDownSuite() {
	testutils.TearDownDB(suite.db)
}

func (suite *ProcessWhitelistStoreTestSuite) getAndCompare(whitelist *storage.ProcessWhitelist) {
	gotWhitelist, _, err := suite.store.Get(whitelist.Id)
	suite.NoError(err)
	suite.Equal(whitelist, gotWhitelist)
}

func (suite *ProcessWhitelistStoreTestSuite) createAndStore() *storage.ProcessWhitelist {
	whitelist := fixtures.GetProcessWhitelistWithID()
	suite.NotNil(whitelist)
	err := suite.store.Upsert(whitelist)
	suite.NoError(err)
	return whitelist
}

func (suite *ProcessWhitelistStoreTestSuite) TestGetNonExistant() {
	whitelist, _, err := suite.store.Get("Not an ID")
	suite.NoError(err)
	suite.Nil(whitelist)
}

func (suite *ProcessWhitelistStoreTestSuite) TestAddGetDeleteWhitelist() {
	whitelist := suite.createAndStore()

	suite.getAndCompare(whitelist)

	err := suite.store.Delete(whitelist.Id)
	suite.NoError(err)
	gotWhitelist, _, err := suite.store.Get(whitelist.Id)
	suite.NoError(err)
	suite.Nil(gotWhitelist)
}

func (suite *ProcessWhitelistStoreTestSuite) TestUpdateWhitelist() {
	whitelist := suite.createAndStore()

	whitelist.Elements = []*storage.WhitelistElement{fixtures.GetWhitelistElement("JosephRules")}
	err := suite.store.Upsert(whitelist)
	suite.NoError(err)

	suite.getAndCompare(whitelist)
}
