package datastore

import (
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/processwhitelist/index"
	whitelistSearch "github.com/stackrox/rox/central/processwhitelist/search"
	"github.com/stackrox/rox/central/processwhitelist/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/set"
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

func (suite *ProcessWhitelistDataStoreTestSuite) testUpdate(key *storage.ProcessWhitelistKey, addProcesses []string, removeProcesses []string, auto bool, expectedResults set.StringSet) *storage.ProcessWhitelist {
	updated, err := suite.datastore.UpdateProcessWhitelistElements(key, fixtures.MakeElements(addProcesses), fixtures.MakeElements(removeProcesses), auto)
	suite.NoError(err)
	suite.NotNil(updated)
	suite.NotNil(updated.Elements)
	suite.Equal(expectedResults.Cardinality(), len(updated.Elements))
	actualResults := set.NewStringSet()
	for _, process := range updated.Elements {
		actualResults.Add(process.GetElement().GetProcessName())
	}
	suite.Equal(expectedResults, actualResults)
	return updated
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
	suite.Equal(updatedWhitelist, gotWhitelist)
}

func (suite *ProcessWhitelistDataStoreTestSuite) TestRoxLockAndUnlockWhitelist() {
	whitelist := suite.createAndStoreWhitelistWithRandomKey()
	key := whitelist.GetKey()
	suite.NotNil(whitelist.GetStackRoxLockedTimestamp())
	// Test that current time is before the StackRox locked time
	suite.True(types.TimestampNow().Compare(whitelist.GetStackRoxLockedTimestamp()) < 0)

	updatedWhitelist, err := suite.datastore.RoxLockProcessWhitelist(key, false)
	suite.NoError(err)
	suite.Nil(updatedWhitelist.GetStackRoxLockedTimestamp())
	gotWhitelist, err := suite.datastore.GetProcessWhitelist(key)
	suite.NoError(err)
	suite.Equal(updatedWhitelist, gotWhitelist)

	updatedWhitelist, err = suite.datastore.RoxLockProcessWhitelist(key, true)
	suite.NoError(err)
	suite.NotNil(updatedWhitelist.GetStackRoxLockedTimestamp())
	// Test that current time is after or equal to the StackRox locked time.
	suite.True(types.TimestampNow().Compare(updatedWhitelist.GetStackRoxLockedTimestamp()) >= 0)
	gotWhitelist, err = suite.datastore.GetProcessWhitelist(key)
	suite.NoError(err)
	suite.Equal(updatedWhitelist, gotWhitelist)
}

func (suite *ProcessWhitelistDataStoreTestSuite) TestUpdateProcessWhitelist() {
	whitelist := fixtures.GetProcessWhitelistWithKey()
	whitelist.Elements = nil // Fixture gives a single process but we want to test updates
	suite.NotNil(whitelist)
	key := whitelist.GetKey()
	id, err := suite.datastore.AddProcessWhitelist(whitelist)
	suite.NoError(err)
	suite.NotNil(id)

	processName := []string{"Some process name"}
	processNameSet := set.NewStringSet(processName...)
	otherProcess := []string{"Some other process"}
	otherProcessSet := set.NewStringSet(otherProcess...)
	updated := suite.testUpdate(key, processName, nil, true, processNameSet)
	suite.True(updated.Elements[0].Auto)

	updated = suite.testUpdate(key, processName, nil, false, processNameSet)
	suite.False(updated.Elements[0].Auto)

	updated = suite.testUpdate(key, otherProcess, processName, true, otherProcessSet)
	suite.True(updated.Elements[0].Auto)

	multiAdd := []string{"a", "b", "c"}
	multiAddExpected := set.NewStringSet(multiAdd...)
	updated = suite.testUpdate(key, multiAdd, otherProcess, false, multiAddExpected)
	for _, process := range updated.Elements {
		suite.False(process.Auto)
	}
}
